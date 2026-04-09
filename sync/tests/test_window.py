from datetime import UTC, datetime, timedelta

import pytest

from acmrank_sync import SyncWindow


def test_sync_window_rejects_inverted_range() -> None:
    start = datetime(2026, 4, 9, 0, 0, tzinfo=UTC)

    with pytest.raises(ValueError):
        SyncWindow(start=start, end=start - timedelta(minutes=1))


def test_sync_window_reports_duration_and_membership() -> None:
    start = datetime(2026, 4, 9, 0, 0, tzinfo=UTC)
    end = start + timedelta(hours=2)
    window = SyncWindow(start=start, end=end)

    assert window.duration_seconds == 7200
    assert window.contains(start + timedelta(minutes=30))
    assert not window.contains(end + timedelta(seconds=1))
