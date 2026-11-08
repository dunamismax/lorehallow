# Owner to-do

Steps only the owner can do: they need the owner physically, money, a new account or a credential agents don't have. Every approval and decision is already pre-granted (AGENTS.md section 2), so nothing here is an approval. Agents build everything up to the last step, write that step here, and keep working; none of these block anything.

Each entry says which BUILD.md item it is, what's ready, and exactly what's left. Agents remove an entry once it's done and the BUILD.md box is checked.

## Open

### 1.1 Bootstrap script and agent SSH keys

- **BUILD.md:** 1.1, "Review and run the bootstrap script with sudo ... and authorize SSH keys for agent sessions on the `deploy` user."
- **Ready:** `deploy/bootstrap/host.sh` exists for fresh hosts. The current host is already hardened by the owner and must not run it ([deploy/HOST.md](../../deploy/HOST.md)). Agents deploy as `root@ubuntu-vm` over Tailscale SSH in check mode.
- **Left for the owner:** nothing to run on this host. When a Tailscale SSH check-mode prompt appears for an agent's deploy session, approve it in the browser. If agents should deploy without that prompt, add a Tailscale SSH rule for this machine in `accept` mode (optional; agents will not change it).
