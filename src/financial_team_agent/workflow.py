import json
import os
import resend
import httpx

from tempfile import NamedTemporaryFile
from workflows import Workflow, step, Context
from workflows.resource import Resource
from typing import Annotated, cast, Any
from llama_index.core.llms.structured_llm import StructuredLLM
from llama_index.core.base.llms.types import ChatMessage
from llama_cloud.types import ClassifierRule
from llama_cloud_services.beta.classifier.client import ClassifyClient
from llama_cloud_services.extract import LlamaExtract, ExtractConfig
from llama_cloud.types import ExtractRun
from .events import EmailReceived, SendEmail, ClassificationResult, EmailProcessed, OutputEvent
from .resources import Invoice, Expense, get_classify_client, get_extract_client, get_llm, WorkflowState, EmailBody

class FinanceTeamAgent(Workflow):
    @step
    async def process_received_email(self, ev: EmailReceived, ctx: Context[WorkflowState]):
        async with ctx.store.edit_state() as state:
            state.sender = ev.sender
            state.subject = ev.subject
            resend.api_key = os.getenv("RESEND_API_KEY")
            attachments = resend.Emails.Receiving.Attachments.list(
                email_id=ev.email_id
            )
            email_data = resend.Emails.Receiving.get(email_id=ev.email_id)
            state.body = email_data["html"] or "No body"
        if attachments["data"]:
            attachment = attachments["data"][0]
            if attachment["content_type"] in ("application/pdf", "image/png", "image/jpg"):
                async with httpx.AsyncClient() as client:
                    response = await client.get(attachment["download_url"]) # type: ignore
                    if response.status_code != 200:
                        return SendEmail(error="An error occurred while downloading the attachment")
                    else:
                        content = response.content
                        fl = NamedTemporaryFile(suffix="." + attachment["content_type"].split("/")[1], delete_on_close=False, delete=False)
                        with open(fl.name, "wb") as f:
                            f.write(content)
                        async with ctx.store.edit_state() as state:
                            state.temporary_file_path = fl.name
                        return EmailProcessed(attachment_file_path=fl.name)
            else:
                return SendEmail(error="The attachment must be a PDF file or a PNG/JPEG image")
    @step
    async def classify_email(self, ev: EmailProcessed, classifier: Annotated[ClassifyClient, Resource(get_classify_client)], ctx: Context[WorkflowState]) -> ClassificationResult | SendEmail:
        rules = [
            ClassifierRule(
                type="invoice",
                description="This is an invoice for a contract that has to be payed out by the company. It may be forwarded from the partner or employee",
            ),
            ClassifierRule(
                type="expense",
                description="This is an expsense that's been submitted for a business trip that should be payed back to the employee in the next pay out cycle.",
            )
        ]
        classification = await classifier.aclassify(files=ev.attachment_file_path, rules=rules)
        if classification.items[0].result is not None:
            return ClassificationResult(
                classification=classification.items[0].result.type,
                reason=classification.items[0].result.reasoning,
            )
        else:
            return SendEmail(error="It was not possible to classify your documents")

    @step
    async def extract_contents(self, ev: ClassificationResult, extract: Annotated[LlamaExtract, Resource(get_extract_client)], llm: Annotated[StructuredLLM, Resource(get_llm)], ctx: Context[WorkflowState]) -> SendEmail:
        state = await ctx.store.get_state()
        if ev.classification == "expense":
            extracted_data = cast(ExtractRun, await extract.aextract(data_schema=Expense, files=ev.attachment, config=ExtractConfig()))
            if extracted_data.data is not None:
                data = cast(dict[str, Any], extracted_data.data)
                if cast(float, data["amount"]) < 1000.0:
                    res = await llm.achat(messages=[ChatMessage(role="system", content="You are an email writer and formatter. Write the email and produce HTML that represents the body"), ChatMessage(content=f"""Construct an email acknowledging to {state.sender} that their expense of {data["amount"]} for {data["description"]} was accepted and will be payed back in the next payment cycle. Keep in mind that {state.sender} sent you this email: {state.body}""", role="user")])
                    os.remove(state.temporary_file_path)
                    if res.message.content is not None:
                        body = EmailBody.model_validate_json(res.message.content)
                        return SendEmail(body=body.html)
                    else:
                        return SendEmail(error="There was an error while generating the reply email")
                else:
                    res = await llm.achat(messages=[ChatMessage(role="system", content="You are an email writer and formatter. Write the email and produce HTML that represents the body"), ChatMessage(content=f"""Contruct an email the their expense of {data["amount"]} for {data["description"]} exceeds the budget so has been denied. Explain that they can reach out if this seems wrong. Keep in mind that {state.sender} sent you this email: {state.body}""")])
                    os.remove(state.temporary_file_path)
                    if res.message.content is not None:
                        body = EmailBody.model_validate_json(res.message.content)
                        return SendEmail(body=body.html)
                    else:
                        return SendEmail(error="There was an error while generating the reply email")
            else:
                os.remove(state.temporary_file_path)
                return SendEmail(error="There was an error while extracting data for the email")
        else:
            extracted_data = cast(ExtractRun, await extract.aextract(data_schema=Invoice, files=ev.attachment, config=ExtractConfig()))
            if extracted_data.data is not None:
                data = cast(dict[str, Any], extracted_data.data)
                res = await llm.achat(messages=[ChatMessage(role="system", content="You are an email writer and formatter. Write the email and produce HTML that represents the body"), ChatMessage(content=f"""Construct a reply to {ev.email}, that the invoice has been received and give in for on who will be payed and how much based on the info in {json.dumps(extracted_data, indent=4)}. Keep in mind that {state.sender} sent you this email: {state.body}""")])
                os.remove(state.temporary_file_path)
                if res.message.content is not None:
                    body = EmailBody.model_validate_json(res.message.content)
                    return SendEmail(body=body.html)
                else:
                    return SendEmail(error="There was an error while generating the reply email")
            else:
                os.remove(state.temporary_file_path)
                return SendEmail(error="There was an error while extracting data for the email")
            
    @step
    async def send_email(self, ev: SendEmail, ctx: Context[WorkflowState]) -> OutputEvent:
        if ev.body is not None:
            state = await ctx.store.get_state()
            resend.api_key = os.getenv("RESEND_API_KEY")
            params: resend.Emails.SendParams = {
                "from": "Clelia from Financial Team <financial@mail.clelia.dev>",
                "to": [state.sender],
                "subject": f"Re: {state.subject}",
                "html": ev.body,
                "reply_to": "financial@mail.clelia.dev"
            }

            resend.Emails.send(params=params)
            return OutputEvent(success=True)
        else:
            async with httpx.AsyncClient() as client:
                await client.post(url=os.getenv("DISCORD_WEBHOOK_URL", ""), json={"content": ev.error})
            return OutputEvent(success=False)
        
agent = FinanceTeamAgent(timeout=600)
            