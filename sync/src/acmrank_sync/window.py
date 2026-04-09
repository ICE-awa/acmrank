from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True, slots=True)
class SyncWindow:
    start: datetime
    end: datetime

    def __post_init__(self) -> None:
        if self.end < self.start:
            raise ValueError("sync window end must not be earlier than start")

    @property
    def duration_seconds(self) -> int:
        return int((self.end - self.start).total_seconds())

    def contains(self, value: datetime) -> bool:
        return self.start <= value <= self.end
