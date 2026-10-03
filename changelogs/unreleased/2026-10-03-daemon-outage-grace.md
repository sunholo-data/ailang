### Fixed — "Notifier subscription DOWN" no longer pages for a drop that reconnects

The notify daemon announced a subscription outage on the first dropped connection, even
when the retry a second later reconnected. A laptop's Pub/Sub pulls break on every sleep,
wake or network change, so Discord received outage alerts for outages that had already
healed. An outage is now announced only once it has lasted 2 minutes, and still only once
per outage. A drop that reconnects is logged. The alert now says how long the
subscription has been down.
