# Reference: legal_obligation_engine — deontic contract reasoner.
# All rules pinned; divergence variants prove each trap discriminates.
import sys

def run(variant="ref"):
    out = []
    # --- contract terms (initial) ---
    ms = {
        "M1": {"deadline": 10, "price": 10000},
        "M2": {"deadline": 40, "price": 20000},
        "M3": {"deadline": 70, "price": 15000},
    }
    PEN_PER_DAY, PEN_CAP = 200, 4000
    PAY_WITHIN, CURE_DAYS = 10, 15
    RATE_PCT, PERIOD = 1, 10

    # --- events (at most one per day) ---
    events = [
        (8,  "deliver", "M1"),
        (30, "amend_price", "M2", 22000),
        (35, "force_majeure", 38, 44),
        (41, "pay", "M1"),
        (45, "waive", "M1-payment"),
        (50, "deliver", "M2"),
        (61, "notice", "M2-payment"),
        (74, "pay", "M2"),
        (80, "notice", "M3-delivery"),
        (96, "terminate", "Client"),
    ]

    delivered = {}   # m -> day
    paid = {}        # m -> day
    notices = {}     # breach-id -> day
    waived = set()
    fm = None
    term_day = None
    term_grounds = None

    for ev in events:
        day, kind = ev[0], ev[1]
        if kind == "deliver": delivered[ev[2]] = day
        elif kind == "pay": paid[ev[2]] = day
        elif kind == "amend_price": ms[ev[2]]["price"] = ev[3]
        elif kind == "force_majeure":
            fm = (ev[2], ev[3])
            span = ev[3] - ev[2] + 1
            for m in ms:
                cond = (ev[2] <= ms[m]["deadline"] <= ev[3]) if variant != "fm_all" else True
                if cond and m not in delivered:
                    ms[m]["deadline"] += span
        elif kind == "notice": notices[ev[2]] = day
        elif kind == "waive": waived.add(ev[2])
        elif kind == "terminate":
            # valid iff some noticed breach: cure window expired, uncured, unwaived
            for bid, nday in notices.items():
                if bid in waived: continue
                m, what = bid.split("-")
                cured = (paid.get(m) is not None and paid[m] <= nday + CURE_DAYS) if what == "payment" \
                    else (delivered.get(m) is not None and delivered[m] <= nday + CURE_DAYS)
                if not cured and day > nday + CURE_DAYS:
                    term_day, term_grounds = day, bid
                    break

    # --- effective terms ---
    for m in ("M1", "M2", "M3"):
        out.append(f"{m} effective: deadline={ms[m]['deadline']} price={ms[m]['price']}")

    vendor_owes = 0
    client_owes = 0
    lines_breach = {}

    for m in ("M1", "M2", "M3"):
        dl, price = ms[m]["deadline"], ms[m]["price"]
        # delivery
        if m in delivered:
            d = delivered[m]
            late = max(0, d - dl)
            pen = min(PEN_PER_DAY * late, PEN_CAP)
            if not (f"{m}-delivery" in waived and variant != "waiver_no_forgive"):
                vendor_owes += pen
            out.append(f"{m} delivery: DELIVERED day={d} late={late} penalty={pen}")
            # payment
            due = d + PAY_WITHIN
            if m in paid:
                p = paid[m]
                late_p = max(0, p - due)
                periods = late_p // PERIOD if variant != "ceil_interest" else -(-late_p // PERIOD)
                interest = price * RATE_PCT * periods // 100
                if f"{m}-payment" in waived and variant != "waiver_no_forgive":
                    client_owes += 0  # forgiven
                else:
                    client_owes += interest
                out.append(f"{m} payment: PAID day={p} late={late_p} interest={interest}")
            else:
                if term_day:
                    out.append(f"{m} payment: CANCELLED")
        else:
            # undelivered
            if term_day:
                late_days = (term_day - 1) - dl
                pen = min(PEN_PER_DAY * late_days, PEN_CAP) if variant != "no_cap" else PEN_PER_DAY * late_days
                if not (f"{m}-delivery" in waived and variant != "waiver_no_forgive"):
                    vendor_owes += pen
                out.append(f"{m} delivery: CANCELLED late_days={late_days} penalty={pen}")
                out.append(f"{m} payment: CANCELLED")
        # breach resolution lines (only noticed or waived)
        for what in ("delivery", "payment"):
            bid = f"{m}-{what}"
            if bid in waived:
                lines_breach[bid] = f"{m} {what} breach: WAIVED"
            elif bid in notices:
                nday = notices[bid]
                cured = (paid.get(m) is not None and paid[m] <= nday + CURE_DAYS) if what == "payment" \
                    else (delivered.get(m) is not None and delivered[m] <= nday + CURE_DAYS)
                lines_breach[bid] = f"{m} {what} breach: {'CURED' if cured else 'UNCURED'}"

    # splice breach lines in canonical position: after the milestone's payment line
    final = []
    for line in out:
        final.append(line)
        for m in ("M1", "M2", "M3"):
            if line.startswith(f"{m} payment:"):
                for what in ("delivery", "payment"):
                    bid = f"{m}-{what}"
                    if bid in lines_breach:
                        final.append(lines_breach.pop(bid))
    out = final
    if term_day:
        out.append(f"termination: day={term_day} by=Client grounds={term_grounds}")
    out.append(f"vendor_owes={vendor_owes}")
    out.append(f"client_owes={client_owes}")
    if vendor_owes > client_owes:
        out.append(f"net: Vendor pays Client {vendor_owes - client_owes}")
    elif client_owes > vendor_owes:
        out.append(f"net: Client pays Vendor {client_owes - vendor_owes}")
    else:
        out.append("net: settled")
    return "\n".join(out)

ref = run()
print(ref)
print("--- divergence checks ---", file=sys.stderr)
for v in ("ceil_interest", "fm_all", "waiver_no_forgive", "no_cap"):
    print(f"{v}: {'DIVERGES' if run(v) != ref else 'SAME (bad!)'}", file=sys.stderr)
