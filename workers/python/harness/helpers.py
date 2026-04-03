from __future__ import annotations

import logging
import sys

from pythonjsonlogger import jsonlogger

_NAME_TO_LEVEL = {
    "PANIC": logging.FATAL,
    "FATAL": logging.FATAL,
    "ERROR": logging.ERROR,
    "WARN": logging.WARNING,
    "INFO": logging.INFO,
    "DEBUG": logging.DEBUG,
    "NOTSET": logging.NOTSET,
}

def configure_logger(log_level: str, log_encoding: str) -> logging.Logger:
    logger = logging.getLogger()
    logger.handlers.clear()
    handler = logging.StreamHandler(stream=sys.stderr)
    if log_encoding == "json":
        format_str = "%(message)%(levelname)%(name)%(asctime)"
        handler.setFormatter(jsonlogger.JsonFormatter(format_str))
    logger.addHandler(handler)
    logger.setLevel(_NAME_TO_LEVEL[log_level.upper()])
    return logger