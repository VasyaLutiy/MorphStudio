# VPS runbook (PM data, not cards)

Host 188.245.28.11 (Ubuntu 24.04), DNS `themorph.stream` (Cloudflare, apex A record, proxy OFF / DNS only) and `morph.root.sx` → that address. Use `themorph.stream`: `morph.root.sx` is filtered on the operator's laptop network.

## morphd
- systemd unit `/etc/systemd/system/morphd.service`: User=morph, WorkingDirectory `/home/morph/morphd` (`.env` and `api-token`, mode 600), `ExecStart=/home/morph/morphd/morphd`, `Restart=always`, log `/home/morph/morph-logs/morphd.log`.
- Listens on `127.0.0.1:7180` only. State `/home/morph/morphd/state`, projects `/home/morph/morphd/projects`.
- `MORPHD_MORPH_BIN=/home/morph/morphd/morph` → `node /home/morph/morph-bin/dist/cli.js` (a copy of MorphV2 `dist/`, refreshed by hand when MorphV2 moves).
- Update after a merge: `git -C ~/MorphStudio pull --ff-only && go -C ~/MorphStudio build -o ~/morphd/morphd.new ./cmd/morphd && mv ~/morphd/morphd.new ~/morphd/morphd`, then (root) `systemctl restart morphd`.

## Cloudflare Tunnel (since 11.10) — the only way in

- `themorph.stream` is a proxied CNAME to the Cloudflare Tunnel `morphd` (id 00e75ae9-2a6f-4f47-ad40-81b0a81f2d2d,
  remotely managed: the route lives in the Zero Trust dashboard — hostname `themorph.stream` → `http://127.0.0.1:7190`,
  catch-all `http_status:404`).
- `cloudflared` 2026.10.0 from the Cloudflare apt repo; own unit `/etc/systemd/system/cloudflared.service`:
  `cloudflared --no-autoupdate tunnel run --token-file /etc/cloudflared/token` (token file root 0600, never in argv).
- Caddy is now only the local origin of the tunnel, `/etc/caddy/Caddyfile`:
```
http://:7190 {
	bind 127.0.0.1
	@session path_regexp session ^/mcp/[^/]+/session$
	respond @session 404
	reverse_proxy 127.0.0.1:7180 {
		flush_interval -1
		header_up Host {upstream_hostport}
	}
	log {
		output file /var/log/caddy/morph-tunnel.log
	}
}
```
  `http://:7190` (any Host) matters: cloudflared forwards `Host: themorph.stream`, and a site named `http://127.0.0.1:7190`
  matches no request (Caddy answers an empty 200). Port 8080 is the ETHSC dashboard — never use it.
- Nothing listens publicly but ssh (22); no Let's Encrypt any more. The old TLS Caddyfile is `Caddyfile.bak-20261011`
  (rollback: restore it, restart caddy, put back the A record 188.245.28.11 with proxy off).
- Checked 11.10 from the laptop: `/healthz` 200; `/projects` and `/mcp` without a token 401; `/mcp/<p>/session` 404;
  `projects_list` over the laptop's MCP `morph` answers; the direct IP on 80/443 refuses.

## Known: the operator's laptop network drops this host name
10.10: from the laptop, any request carrying the name `morph.root.sx` (TLS SNI or HTTP Host) hangs after the TCP connect, while the same IP with another Host answers and the name works from the VPS and from other networks — filtering on the path, not the server. Until a different name or a VPN: `ssh -i <key> -N -L 7180:127.0.0.1:7180 root@188.245.28.11` and `http://127.0.0.1:7180/mcp`.

## Cron
The morph user's crontab is empty since the cutover (10.10); the old one is in `~/crontab.bak-20261010`.

## Tenant sessions (Gate 0 probe D, 11.10)

- Beta tenants are Linux users `mu-<name>` (home 0700); today `mu-a`, `mu-b` (probe users).
- The claude binary for tenants: `/opt/claude/claude` → `/opt/claude/claude-<version>` (root 0755; the morph user's
  `~/.local` copy is unreadable to tenants). Refresh by hand when claude moves.
- morphd (User=morph) starts a tenant session only through the root wrapper `/usr/local/sbin/morph-spawn` (repo
  `tools/morph-spawn`, install `install -o root -g root -m 755`), allowed by `/etc/sudoers.d/morph-spawn` (0440):
  `morph ALL=(root) NOPASSWD: /usr/local/sbin/morph-spawn, /usr/local/sbin/morph-run`. The morph-spawn wrapper fixes every systemd property (uid/gid,
  PrivateTmp, MemoryMax 3G, CPUQuota 150%, TasksMax 512, ProtectSystem=strict, ProtectHome=tmpfs + BindPaths of the
  tenant home, NoNewPrivileges) and validates the tenant, the workdir (inside the tenant home) and the files (under
  `/home/morph/morphd/state/`, owned by morph, 0600). The token file becomes the unit's `EnvironmentFile`, the MCP
  config a systemd credential (`LoadCredential=mcp:` → `--mcp-config /run/credentials/<unit>.service/mcp`).
- Why not capabilities: `AmbientCapabilities=CAP_SETUID CAP_SETGID` let morph `setpriv` to a tenant but not run
  `systemd-run` (polkit: "Interactive authentication required"), so no cgroup limits or sandbox — rejected.
- `/usr/local/sbin/morph-run` (repo `tools/morph-run`, probe E): one `git` or `morph` command for a tester's project under
  the tester's uid in the same sandbox (no token, `RuntimeMaxSec=600`); `--git-credentials <file under state, morph 0600>`
  installs the tester's `~/.git-credentials` (0600) and sets `credential.helper store` for the tester.
- The morph binary for testers: `/opt/morph` (copy of `~/morph-bin` with `node_modules` and `templates` dereferenced,
  root, read-only) and `/opt/morph/morph` (`exec node /opt/morph/dist/cli.js "$@"`). Refresh by hand when MorphV2 moves.
