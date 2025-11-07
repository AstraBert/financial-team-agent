# Financial Team Agent

Are you also drowning in invoices and expenses that your employees are sending you? Well, with the Financial Team Agent, you won't be anymore.

Once you set it up, all the emails coming to your inbox containing attachments will be:

- Classified as either carrying an invoice or an expense through [LlamaClassify](https://developers.llamaindex.ai/python/cloud/llamaclassify/getting_started/)
- Processed by [LlamaExtract](https://developers.llamaindex.ai/python/cloud/llamaextract/getting_started/) to obtain the relevant information 
- Automatically replied to, using OpenAI GPT-4.1 and [Resend](https://resend.com)

## Set Up

**Clone this repository:**

```bash
git clone https://github.com/AstraBert/financial-team-agent
cd financial-team-agent
```

**Deploy the LlamaAgent:**

In order for the [agent workflow](./src/financial_team_agent/workflow.py) to receive and process emails, it needs to be deployed to the cloud (or at least accessible through a public endpoint). The easiest way to do so is to use [`llamactl`](https://developers.llamaindex.ai/python/llamaagents/llamactl/getting-started/) and deploy the agent workflow as a [LlamaAgent](https://developers.llamaindex.ai/python/llamaagents/overview/):

```bash
uv tool install -U llamactl
llamactl auth # authenticate
llamactl deployments create # create a deployment from the current repository
```

In order for the LlamaAgent to work, you will need the following environment variables in a `.env` file (`llamactl` manages environments autonomously):

- `OPENAI_API_KEY` to interact with GPT-4.1 for email generation
- `LLAMA_CLOUD_API_KEY` to get predictions from LlamaClassify and LlamaExtract
- `DISCORD_WEBHOOK_URL` to get Discord notifications in case there are failures during the workflow execution ([guide on how to set up Discord webhooks](https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks))

Once the agent is deployed, build the Docker image for the webhook (needed for Resend to receive emails), and deploy it through services like [Dokploy](https://dokploy.com) or [Coolify](https://coolify.io).

```bash
docker build . -t your-username/resend-webhook:prod
# docker login ghcr.io # (uncomment if you wish to use the GitHub container registry)
docker push your-username/resend-webhook:prod
```

The webhook service uses a few env variables:

- `LLAMA_CLOUD_API_KEY` and `LLAMA_CLOUD_API_ENDPOINT`, the API key and the API endpoint to interact with your deployed LlamaAgent
- `POSTHOG_API_KEY` and `POSTHOG_ENDPOINT` for monitoring (not needed, if they are not set there will be no monitoring)

Services like Dokploy or Coolify offer you to set these environment variables through their own environment management interfaces.

Connect your webhook endpoint (in the form of `https://your.domain.com/webhook`) to the [Resend webhooks service](https://resend.com/docs/dashboard/receiving/introduction#how-does-it-work).

## Use

Try the agent by sending an email to your `.resend.app` email address (such as financial@<your-id>.resend.app), attaching either an invoice or an expense as a PDF or PNG/JPEG file.

In a couple of minutes, you will receive the email back, and you can check the progress of this through the webhook page on your Resend dashboard, or through Posthog events if you enabled them.

## License

This project is provided under [MIT License](./LICENSE)