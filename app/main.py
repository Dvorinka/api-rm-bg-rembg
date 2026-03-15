import base64
import hmac
import io
import os
import time
from typing import Optional

from fastapi import Depends, FastAPI, File, Header, HTTPException, UploadFile
from pydantic import BaseModel
from PIL import Image

try:
    from rembg import remove as rembg_remove
    REMBG_IMPORT_ERROR: Optional[str] = None
except Exception as exc:  # pragma: no cover - service startup fallback
    rembg_remove = None
    REMBG_IMPORT_ERROR = str(exc)


app = FastAPI(title="Background Removal API", version="1.0.0")


def expected_api_key() -> str:
    return os.getenv("RMBG_API_KEY", "dev-rmbg-key")


def authorize(
    x_api_key: Optional[str] = Header(default=None, alias="X-API-Key"),
    authorization: Optional[str] = Header(default=None, alias="Authorization"),
) -> None:
    key = (x_api_key or "").strip()
    if not key and authorization:
        lower = authorization.lower().strip()
        if lower.startswith("bearer "):
            key = authorization[7:].strip()

    if not hmac.compare_digest(key, expected_api_key()):
        raise HTTPException(status_code=401, detail="unauthorized")


class Base64Input(BaseModel):
    file_base64: str


def normalize_to_png(raw_image_bytes: bytes) -> bytes:
    img = Image.open(io.BytesIO(raw_image_bytes)).convert("RGBA")
    out = io.BytesIO()
    img.save(out, format="PNG")
    return out.getvalue()


def remove_background(raw_image_bytes: bytes) -> bytes:
    if rembg_remove is None:
        raise HTTPException(
            status_code=503,
            detail="rembg runtime unavailable; install dependencies from rm-bg-rembg/requirements.txt",
        )
    try:
        output = rembg_remove(raw_image_bytes)
        return normalize_to_png(output)
    except Exception as exc:  # pragma: no cover - defensive service boundary
        raise HTTPException(status_code=400, detail=f"background removal failed: {exc}") from exc


@app.get("/healthz")
def healthz() -> dict:
    payload: dict = {"status": "ok"}
    if REMBG_IMPORT_ERROR:
        payload["status"] = "degraded"
        payload["warning"] = "rembg import failed"
        payload["detail"] = REMBG_IMPORT_ERROR
    return payload


@app.post("/v1/rmbg/remove")
async def remove_file(
    _: None = Depends(authorize),
    file: UploadFile = File(...),
) -> dict:
    content = await file.read()
    if not content:
        raise HTTPException(status_code=400, detail="file is empty")

    started = time.perf_counter()
    result_png = remove_background(content)
    elapsed_ms = int((time.perf_counter() - started) * 1000)

    return {
        "data": {
            "filename": file.filename or "input",
            "output_filename": "output.png",
            "output_mime": "image/png",
            "size_bytes": len(result_png),
            "processing_ms": elapsed_ms,
            "output_base64": base64.b64encode(result_png).decode("ascii"),
        }
    }


@app.post("/v1/rmbg/remove/base64")
def remove_base64(
    payload: Base64Input,
    _: None = Depends(authorize),
) -> dict:
    try:
        content = base64.b64decode(payload.file_base64, validate=True)
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"invalid file_base64: {exc}") from exc

    if not content:
        raise HTTPException(status_code=400, detail="file_base64 is empty")

    started = time.perf_counter()
    result_png = remove_background(content)
    elapsed_ms = int((time.perf_counter() - started) * 1000)

    return {
        "data": {
            "output_filename": "output.png",
            "output_mime": "image/png",
            "size_bytes": len(result_png),
            "processing_ms": elapsed_ms,
            "output_base64": base64.b64encode(result_png).decode("ascii"),
        }
    }
