from __future__ import annotations

from aa_sifter.context.handoff import SecretRedactor


def test_redacts_env_assignments():
    redactor = SecretRedactor()
    text = "DATABASE_URL=postgres://user:pass@host/db\nNORMAL_VALUE=hello\n"
    result = redactor.redact(text)
    assert "postgres://user:pass@host/db" not in result.text
    assert "NORMAL_VALUE=hello" in result.text


def test_redacts_common_api_keys():
    redactor = SecretRedactor()
    text = "key=sk-abcdefghijklmnopqrstuvwxyz token=ghp_abcdefghijklmnopqrstuvwxyz"
    result = redactor.redact(text)
    assert "sk-abcdefghijklmnopqrstuvwxyz" not in result.text
    assert "ghp_abcdefghijklmnopqrstuvwxyz" not in result.text
    assert result.redacted


def test_redacts_private_keys_and_bearer_tokens():
    redactor = SecretRedactor()
    text = (
        "-----BEGIN RSA PRIVATE KEY-----\nSECRETMATERIAL\n-----END RSA PRIVATE KEY-----\n"
        "Authorization: Bearer abcdefghijklmnop1234"
    )
    result = redactor.redact(text)
    assert "SECRETMATERIAL" not in result.text
    assert "abcdefghijklmnop1234" not in result.text


def test_redact_messages_collects_findings():
    redactor = SecretRedactor()
    messages = [
        {"role": "system", "content": "you are helpful"},
        {"role": "user", "content": "password = hunter2secret"},
    ]
    redacted, findings = redactor.redact_messages(messages)
    assert "hunter2secret" not in redacted[1]["content"]
    assert findings


def test_redaction_can_be_disabled():
    redactor = SecretRedactor(enabled=False)
    result = redactor.redact("password = hunter2secret")
    assert "hunter2secret" in result.text
    assert result.redacted is False
