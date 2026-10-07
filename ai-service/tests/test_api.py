import io

import pytest
from fastapi.testclient import TestClient
from reportlab.pdfgen import canvas

from app.main import app, settings


@pytest.fixture
def client():
    with TestClient(app) as c:
        yield c


def make_pdf(lines: list[str]) -> bytes:
    buf = io.BytesIO()
    c = canvas.Canvas(buf)
    y = 800
    for line in lines:
        c.drawString(50, y, line)
        y -= 20
    c.save()
    return buf.getvalue()


def test_healthz(client):
    assert client.get("/healthz").json()["provider"] == "mock"


def test_analyze_text_returns_camel_case(client):
    r = client.post("/v1/analyze/text", json={"text": "보증금 1억원\n제1조 임대인은 근저당을 설정하지 않는다.\n제2조 임차인은 모든 시설물을 원상복구한다."})
    assert r.status_code == 200
    body = r.json()
    assert {"riskScore", "lossAmount", "lossRange", "clauses", "modelVersion", "policyVersion"} <= body.keys()
    assert body["clauses"][0]["clauseType"] in ("선순위근저당", "원상복구과도부담")


def test_analyze_pdf_text_layer(client):
    pdf = make_pdf(["Special terms of lease agreement", "1. The tenant shall pay all repair costs of the premises."])
    r = client.post("/v1/analyze", files={"file": ("c.pdf", pdf, "application/pdf")})
    assert r.status_code == 200
    assert r.json()["stats"]["clauses"] >= 1


def test_rejects_non_pdf(client):
    r = client.post("/v1/analyze", files={"file": ("a.txt", b"hello", "text/plain")})
    assert r.status_code == 415


def test_rejects_pdf_without_text(client):
    buf = io.BytesIO()
    c = canvas.Canvas(buf)
    c.rect(10, 10, 50, 50)
    c.save()
    r = client.post("/v1/analyze", files={"file": ("scan.pdf", buf.getvalue(), "application/pdf")})
    assert r.status_code == 422 and "스캔" in r.json()["detail"]


def test_token_required_when_configured(client, monkeypatch):
    monkeypatch.setattr(settings, "internal_token", "secret")
    assert client.post("/v1/analyze/text", json={"text": "x" * 20}).status_code == 401
    ok = client.post("/v1/analyze/text", json={"text": "x" * 20}, headers={"X-Internal-Token": "secret"})
    assert ok.status_code == 200


def test_stream_emits_progress_then_result(client):
    import json

    pdf = make_pdf(["Special terms of lease agreement", "1. The tenant shall pay all repair costs of the premises.", "2. The tenant shall not renew."])
    with client.stream("POST", "/v1/analyze/stream", files={"file": ("c.pdf", pdf, "application/pdf")}) as r:
        assert r.status_code == 200
        events = [json.loads(line) for line in r.iter_lines() if line]
    assert events[0] == {"type": "progress", "stage": "reading", "done": 0, "total": 0}
    stages = [e["stage"] for e in events if e["type"] == "progress"]
    assert stages[:3] == ["reading", "splitting", "classifying"] and stages[-1] == "scoring"
    done = [e["done"] for e in events if e["type"] == "progress" and e["stage"] == "classifying"]
    assert done == sorted(done) and done[-1] == events[-1]["data"]["stats"]["clauses"]
    assert events[-1]["type"] == "result" and "riskScore" in events[-1]["data"]


def test_stream_rejects_bad_file_before_streaming(client):
    r = client.post("/v1/analyze/stream", files={"file": ("a.txt", b"hello", "text/plain")})
    assert r.status_code == 415
