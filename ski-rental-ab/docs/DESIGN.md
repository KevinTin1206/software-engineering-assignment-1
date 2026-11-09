# Approved rental prototype design

Approved in chat on 2026-11-01, including identity image uploads.

Two independently runnable Go/HTML/JS projects: ski (a+b), bicycle (c).
Go standard library only. Thread-safe memory, no database. German UI.
One Go server serves static assets and JSON API; `go run .` starts it.
Business structs/methods, store and HTTP handlers have separate responsibilities.

Equipment and people CRUD; reservations, checkout, return and cancellation;
Dashboard: available now, physically rented, total equipment and people.
Inclusive calendar dates (same day = one day). Reservations block overlapping
inclusive intervals; checkout is only allowed during the reservation interval
and when no active physical loan exists. No new backdated reservations.
Returning releases the equipment; cancelling applies only to reservations.
Referenced equipment/people cannot be deleted. Equipment with rental history
cannot change category, preserving the meaning of recorded quotes.

Editable category daily rates plus minimum-duration daily rate tiers, longest
eligible duration wins. EUR amounts in integer cents; confirmed rentals keep
price snapshots. No payment, seasonal pricing, authentication or plugins.

Optional one JPG/PNG identity image per person, at most 5 MiB; preview, replace,
delete. Validate actual image encoding; memory storage and restricted image
endpoint, no identity uploads in static web folder. Demonstrate with fake data.

Reuse the same small domain/store/handler design by copying it into each ZIP;
each has its own go.mod, web assets, demo seed, README and local Git repository.
No outside shared-module dependency. Compare this with separate reimplementation
(duplicate defects) and an external shared framework (harder standalone setup).

Unknown from screenshots: exact classroom fields, precise deadline time,
mandatory GitHub submission, mandatory upload timing, framework/d requirements.
Current scope follows the approved chat plan rather than claiming these resolved.
