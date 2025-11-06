from workflows.events import StartEvent, StopEvent, Event

class EmailReceived(StartEvent):
    sender: str
    subject: str
    email_id: str

class EmailProcessed(Event):
    attachment_file_path: str

class ClassificationResult(Event):
    classification: str
    reason: str

class SendEmail(Event):
    body: str | None = None
    error: str | None = None

class OutputEvent(StopEvent):
    success: bool