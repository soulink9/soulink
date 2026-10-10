// English copy (/en/ path). Shape must mirror zh.ts.
import type zh from './zh';

const en: typeof zh = {
  locale: 'en',
  altPath: '/',
  altLabel: '中文',
  meta: {
    title: 'soulink — Share local services with one command',
    desc: 'soulink is an open-source tunneling tool: one command to install, one command to expose a local port as a public HTTPS URL. Anonymous by default, register for a fixed subdomain.',
  },
  nav: { features: 'Features', quickstart: 'Quickstart', tiers: 'Tiers', github: 'GitHub' },
  hero: {
    badge: 'Open source · MIT · No sign-up',
    titleA: 'One command to',
    titleB: 'share your localhost with the world',
    sub: 'No account, no config. Install soulink, run one command, and your local port becomes a public HTTPS URL.',
    ctaStart: 'Quickstart',
    ctaGithub: 'GitHub',
    termComment1: '# install (macOS / Linux)',
    termComment2: '# expose local port 3000',
    termNote: 'Anonymous · random subdomain · time-limited',
  },
  features: {
    title: 'Features',
    items: [
      { t: 'Anonymous by default', d: 'No sign-up, no login. Run soulink 3000 and get a random public URL instantly.' },
      { t: 'Fixed subdomains', d: 'Register to claim a durable subdomain like myapp.apps.soulink.cam.' },
      { t: 'Single binary', d: 'Written in Go, zero dependencies. macOS / Linux / Windows, one-command install.' },
      { t: 'Quotas & rate limits', d: 'Per-account rate limiting and daily/monthly traffic quotas, with fair-share protection for anonymous tunnels.' },
      { t: 'Built-in support', d: 'Message the admin right from the console — no issue tracker detour.' },
      { t: 'Open & self-hostable', d: 'MIT licensed. Both hub and client are open source; run your own hub.' },
    ],
  },
  quickstart: {
    title: 'Quickstart',
    s1t: '1. Install',
    s1d: 'One-command install (macOS / Linux):',
    s1note: 'The installer falls back to mirrors automatically; set SOULINK_GITHUB_MIRROR=<host> to pin one.',
    s2t: '2. Start a tunnel (anonymous)',
    s2d: 'Expose local port 3000 with a random subdomain:',
    s3t: '3. Claim a fixed subdomain (optional)',
    s3d: 'Register on the hub, copy your token, then configure the client:',
    winD: 'Windows (PowerShell 5.1+), one command:',
    winZip: 'Or grab the zip from Releases and run it directly.',
  },
  tiers: {
    title: 'Tiers & quotas',
    desc: 'Anonymous tunnels work out of the box; an account unlocks higher quotas and fixed subdomains.',
    cols: ['Tier', 'Subdomain', 'Concurrent tunnels', 'Bandwidth', 'Traffic quota'],
    rows: [
      ['guest (anon)', 'Random, reclaimed after 4h', 'Shared pool of 20', '256 KB/s shared', '200 MB/day per tunnel'],
      ['free', 'Custom subdomain', '3 (random ≤ 2)', '512 KB/s', '100 MB/day · 1 GB/month'],
      ['pro', '3 reserved subdomains', '10', '1 MB/s', '5 GB/day · 100 GB/month'],
      ['max', '10 reserved subdomains', '50', '5 MB/s', '100 GB/day · 1 TB/month'],
    ],
    note: 'Quotas follow hub configuration; the above are the current defaults of soulink.cam.',
  },
  footer: {
    tagline: 'Open-source tunneling. One command to go live.',
    links: 'Links',
    github: 'GitHub',
    releases: 'Releases',
    console: 'Console',
    license: 'MIT License',
  },
};
export default en;
