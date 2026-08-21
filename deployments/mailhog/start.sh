#!/bin/sh
# Bind MailHog's HTTP UI/API to Render's PORT; SMTP stays on 1025 for the API.
set -eu
export MH_UI_BIND_ADDR="0.0.0.0:${PORT:-8025}"
export MH_API_BIND_ADDR="0.0.0.0:${PORT:-8025}"
export MH_SMTP_BIND_ADDR="0.0.0.0:1025"
export MH_HOSTNAME="${MH_HOSTNAME:-clap-mailhog}"
exec MailHog
