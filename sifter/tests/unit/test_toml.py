from __future__ import annotations

from aa_sifter.config.toml import dumps


def test_dumps_scalars_and_tables():
    text = dumps(
        {
            "version": 1,
            "active": "default",
            "enabled": True,
            "ratio": 0.5,
            "tags": ["a", "b"],
            "profile": {"local": {"provider": "ollama", "model": "x"}},
        }
    )
    assert "version = 1" in text
    assert "enabled = true" in text
    assert 'tags = ["a", "b"]' in text
    assert "[profile.local]" in text
    assert 'provider = "ollama"' in text


def test_dumps_array_of_tables():
    text = dumps(
        {
            "profile": {
                "local_fallbacks": [
                    {"provider": "ollama", "model": "a"},
                    {"provider": "ollama", "model": "b"},
                ]
            }
        }
    )
    assert text.count("[[profile.local_fallbacks]]") == 2


def test_dumps_options_table():
    text = dumps({"profile": {"local": {"options": {"num_gpu": 1}}}})
    assert "[profile.local.options]" in text
    assert "num_gpu = 1" in text


def test_dumps_escapes_strings():
    text = dumps({"value": 'a "quoted" path\\x'})
    assert '\\"' in text
