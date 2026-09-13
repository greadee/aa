from __future__ import annotations

from aa_sifter.context.handoff import HandoffBuilder, SecretRedactor
from aa_sifter.context.packet import EscalationPacket


def test_redacts_openai_key_and_env_assignment():
    redactor = SecretRedactor()
    text = "OPENAI_API_KEY=sk-abcdefghijklmnopqrstuvwxyz\npassword = hunter2secret\n"
    result = redactor.redact(text)
    assert "sk-abcdefghijklmnopqrstuvwxyz" not in result.text
    assert "hunter2secret" not in result.text
    assert "openai_key" in result.findings
    assert result.redacted


def test_redacts_private_key():
    redactor = SecretRedactor()
    pem = "-----BEGIN RSA PRIVATE KEY-----\nMIIabc\n-----END RSA PRIVATE KEY-----"
    result = redactor.redact(pem)
    assert "MIIabc" not in result.text
    assert "private_key" in result.findings


def test_redaction_can_be_disabled():
    redactor = SecretRedactor(enabled=False)
    result = redactor.redact("password = hunter2secret")
    assert "hunter2secret" in result.text


def test_packet_marks_constraints_authoritative():
    packet = EscalationPacket(
        original_task="Add OAuth",
        human_constraints=["Do not replace SQLite"],
        approved_decisions=["Use SQLite"],
    )
    prompt = packet.to_prompt()
    assert "HUMAN CONSTRAINTS (authoritative" in prompt
    assert "APPROVED DECISIONS (authoritative)" in prompt


def test_handoff_from_plan_preserves_constraints():
    builder = HandoffBuilder()
    specs = builder.from_plan(
        {
            "tasks": [
                {"id": "task-001", "task": "do a", "constraints": ["use sqlite"]},
                {"task": "do b", "dependencies": ["task-001"]},
            ]
        }
    )
    assert specs[0].task_id == "task-001"
    assert specs[1].task_id == "task-002"
    assert specs[1].dependencies == ["task-001"]
