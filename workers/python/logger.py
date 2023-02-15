import json
import logging
import traceback
from datetime import datetime


class JsonEncoderStrFallback(json.JSONEncoder):
    def default(self, obj):
        try:
            return super().default(obj)
        except TypeError as exc:
            if "not JSON serializable" in str(exc):
                return str(obj)
            raise


class JsonEncoderDatetime(JsonEncoderStrFallback):
    def default(self, obj):
        if isinstance(obj, datetime):
            return obj.strftime("%Y-%m-%dT%H:%M:%S%z")
        else:
            return super().default(obj)


class JsonLogRecord(logging.LogRecord):
    json_formatted: str


def json_record_factory(*args, **kwargs) -> logging.LogRecord:
    record = JsonLogRecord(*args, **kwargs)

    record.json_formatted = json.dumps(
        {
            "level": record.levelname,
            "ts": record.created,
            "location": "{}:{}:{}".format(
                record.pathname or record.filename,
                record.funcName,
                record.lineno,
            ),
            "exception": record.exc_info,
            "traceback": (
                traceback.format_exception(*record.exc_info)
                if record.exc_info
                else None
            ),
            "msg": record.getMessage(),
        },
        cls=JsonEncoderDatetime,
    )
    record.exc_info = None
    record.exc_text = None

    return record
