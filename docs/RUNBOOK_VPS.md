# VPS runbook (PM data, not cards)

Host 188.245.28.11 (Ubuntu 24.04), DNS `morph.root.sx` and `morph.themorph.stream` (Cloudflare, A record, DNS only) → that address.

## morphd
- systemd unit `/etc/systemd/system/morphd.service`: User=morph, WorkingDirectory `/home/morph/morphd` (`.env` and `api-token`, mode 600), `ExecStart=/home/morph/morphd/morphd`, `Restart=always`, log `/home/morph/morph-logs/morphd.log`.
- Listens on `127.0.0.1:7180` only. State `/home/morph/morphd/state`, projects `/home/morph/morphd/projects`.
- `MORPHD_MORPH_BIN=/home/morph/morphd/morph` → `node /home/morph/morph-bin/dist/cli.js` (a copy of MorphV2 `dist/`, refreshed by hand when MorphV2 moves).
- Update after a merge: `git -C ~/MorphStudio pull --ff-only && go -C ~/MorphStudio build -o ~/morphd/morphd.new ./cmd/morphd && mv ~/morphd/morphd.new ~/morphd/morphd`, then (root) `systemctl restart morphd`.

## Caddy (TLS in front of morphd)
- Ubuntu package `caddy` 2.6.2, `/etc/caddy/Caddyfile` (the distro default kept as `Caddyfile.dist`):
```
morph.root.sx, morph.themorph.stream {
	@session path_regexp session ^/mcp/[^/]+/session$
	respond @session 404
	reverse_proxy 127.0.0.1:7180 {
		flush_interval -1
	}
	log {
		output file /var/log/caddy/morph.root.sx.log
	}
}
```
- The per-session MCP endpoint `/mcp/<project>/session` is for claude sessions on the host only: 404 from outside.
- Certificate from Let's Encrypt, obtained 10.10.
- Endpoints for the PM: `https://morph.root.sx/mcp` (projects_list, project_create), `https://morph.root.sx/mcp/<project>` (status, plan_load, order, …), HTTP API under `https://morph.root.sx/projects/…`; all need `Authorization: Bearer <MORPHD_TOKEN>`.

## Known: the operator's laptop network drops this host name
10.10: from the laptop, any request carrying the name `morph.root.sx` (TLS SNI or HTTP Host) hangs after the TCP connect, while the same IP with another Host answers and the name works from the VPS and from other networks — filtering on the path, not the server. Until a different name or a VPN: `ssh -i <key> -N -L 7180:127.0.0.1:7180 root@188.245.28.11` and `http://127.0.0.1:7180/mcp`.

## Cron
The morph user's crontab is empty since the cutover (10.10); the old one is in `~/crontab.bak-20261010`.
