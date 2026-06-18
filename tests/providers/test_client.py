from __future__ import annotations

import pytest

from wi.providers.client import resolve_model, _FALLBACK_MODEL, _PROVIDER_DEFAULTS


def test_fallback_when_no_env(monkeypatch):
    monkeypatch.delenv("WI_MODEL", raising=False)
    monkeypatch.delenv("WI_PROVIDER", raising=False)
    assert resolve_model() == _FALLBACK_MODEL


def test_explicit_model_overrides_env(monkeypatch):
    monkeypatch.setenv("WI_PROVIDER", "openai")
    monkeypatch.setenv("WI_MODEL", "gpt-4o-mini")
    assert resolve_model("anthropic/claude-opus-4-8") == "anthropic/claude-opus-4-8"


def test_wi_model_qualified_used_as_is(monkeypatch):
    monkeypatch.setenv("WI_MODEL", "openai/gpt-4o")
    monkeypatch.delenv("WI_PROVIDER", raising=False)
    assert resolve_model() == "openai/gpt-4o"


def test_wi_model_qualified_ignores_provider(monkeypatch):
    monkeypatch.setenv("WI_PROVIDER", "anthropic")
    monkeypatch.setenv("WI_MODEL", "openai/gpt-4o")
    assert resolve_model() == "openai/gpt-4o"


def test_wi_provider_only_uses_default(monkeypatch):
    for provider, expected in _PROVIDER_DEFAULTS.items():
        monkeypatch.setenv("WI_PROVIDER", provider)
        monkeypatch.delenv("WI_MODEL", raising=False)
        assert resolve_model() == expected


def test_wi_provider_and_bare_model_combined(monkeypatch):
    monkeypatch.setenv("WI_PROVIDER", "openai")
    monkeypatch.setenv("WI_MODEL", "gpt-4o-mini")
    assert resolve_model() == "openai/gpt-4o-mini"


def test_unknown_provider_no_model_raises(monkeypatch):
    monkeypatch.setenv("WI_PROVIDER", "notareal")
    monkeypatch.delenv("WI_MODEL", raising=False)
    with pytest.raises(ValueError, match="WI_PROVIDER='notareal'"):
        resolve_model()


def test_unknown_provider_with_bare_model_combined(monkeypatch):
    monkeypatch.setenv("WI_PROVIDER", "notareal")
    monkeypatch.setenv("WI_MODEL", "some-model")
    assert resolve_model() == "notareal/some-model"
