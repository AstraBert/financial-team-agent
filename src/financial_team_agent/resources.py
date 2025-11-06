import os

from llama_cloud_services import LlamaExtract
from llama_cloud.client import AsyncLlamaCloud
from pydantic import BaseModel, Field
from llama_cloud_services.beta.classifier.client import ClassifyClient
from llama_index.llms.openai import OpenAI
from llama_index.core.llms.structured_llm import StructuredLLM

class Expense(BaseModel):
    amount: float = Field(description="The amount of the expense")
    currency: str = Field(description="The currency of the expense")
    description: str = Field(description="A description of the expense")

class Invoice(BaseModel):
    amount: float = Field(description="The amount of the invoice")
    currency: str = Field(description="The currency of the invoice")
    due_date: str = Field(description="The due date of the invoice")
    payee: str = Field(description="The payee of the invoice")

class EmailBody(BaseModel):
    html: str = Field(description="Body of the email represented as HTML")

class WorkflowState(BaseModel):
    sender: str = ""
    subject: str = ""
    body: str = ""
    temporary_file_path: str = ""

llama_extract = LlamaExtract(api_key=os.getenv("LLAMA_CLOUD_API_KEY"))
llama_cloud_client = AsyncLlamaCloud(token=os.getenv("LLAMA_CLOUD_API_KEY"))

async def get_extract_client(*args, **kwargs) -> LlamaExtract:
    return llama_extract

async def get_classify_client(*args, **kwargs) -> ClassifyClient:
    return ClassifyClient(llama_cloud_client)

async def get_llm(*args, **kwargs) -> StructuredLLM:
    return OpenAI(model="gpt-4.1", api_key=os.getenv("OPENAI_API_KEY")).as_structured_llm(EmailBody)