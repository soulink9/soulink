// English copy (/en/ path). Shape must mirror zh.ts.
import type zh from './zh';

const en: typeof zh = {
  locale: 'en',
  altPath: '/',
  altLabel: '中文',
  meta: {
    title: 'Soulink — Share your AI sparks with friends in one command',
    desc: 'Soulink is an open-source tunneling tool: one command to install, one command to share a local port with friends. Anonymous by default, register for a fixed domain.',
  },
  nav: { brand: 'Soulink', quickstart: 'Quickstart', features: 'Features', tiers: 'Tiers', faq: 'FAQ', github: 'GitHub', theme: 'Toggle theme' },
  hero: {
    titleA: 'Share your AI sparks',
    titleB: 'with friends, in one command',
    sub: 'No account, no config. One command turns your local AI apps and services into public HTTPS URLs, ready to share.',
  },
  features: {
    title: 'Features',
    items: [
      { t: 'Anonymous by default', d: 'No sign-up, no login. Run soulink 3000 and get a random public URL instantly.' },
      { t: 'Fixed domains', d: 'Register to claim a fixed domain like myapp.apps.soulink.cam.' },
      { t: 'Single binary', d: 'Written in Go, zero dependencies. macOS / Linux / Windows, one-command install.' },
      { t: 'Quotas & rate limits', d: 'Per-account rate limiting and daily/monthly traffic quotas, with fair-share protection for anonymous tunnels.' },
      { t: 'Built-in support', d: 'Message the admin right from the console — no issue tracker detour.' },
      { t: 'Open-source client', d: 'MIT licensed, fully open on GitHub — transparent and auditable.' },
    ],
  },
  quickstart: {
    s1t: '1. Install',
    copy: 'Copy',
    copied: 'Copied ✓',
    s1note: 'The installer falls back to mirrors automatically; to pin one explicitly:',
    s2t: '2. Start a tunnel (anonymous)',
    s2d: 'Share local port 3000 with friends on a random domain:',
    s3t: '3. Claim a fixed domain (optional)',
    s3d: 'Register in the console, copy your token, then configure the client:',
    winZip: 'Or grab the zip from Releases and run it directly.',
  },
  tiers: {
    title: 'Tiers & quotas',
    desc: 'Anonymous tunnels work out of the box; an account unlocks higher quotas and fixed domains.',
    cols: ['Tier', 'Subdomain', 'Concurrent tunnels', 'Bandwidth', 'Traffic quota'],
    rows: [
      ['guest (anon)', 'Random, reclaimed after 4h', 'Shared pool of 20', '256 KB/s shared', '200 MB/day per tunnel'],
      ['free', 'Custom domain', '3 (random ≤ 2)', '512 KB/s', '100 MB/day · 1 GB/month'],
      ['pro', '3 reserved domains', '10', '1 MB/s', '5 GB/day · 100 GB/month'],
      ['max', '10 reserved domains', '50', '5 MB/s', '100 GB/day · 1 TB/month'],
    ],
    note: 'Quotas follow hub configuration; the above are the current defaults of soulink.cam.',
  },
  faq: {
    title: 'FAQ',
    items: [
      {
        q: 'How is soulink different from ngrok or frp?',
        a: 'Open source (MIT) and anonymous by default: one install command, one command to get a public HTTPS URL — no account needed. Register to claim a fixed domain. Single-binary client for macOS / Linux / Windows.',
      },
      {
        q: 'What are the limits of anonymous usage?',
        a: 'Guest tunnels need no account: random domain (<code>xxx.apps.soulink.cam</code>), reclaimed after 4 hours, site-wide shared concurrency and bandwidth, 200 MB/day per tunnel. Great for quick demos; register for fixed domains and higher quotas (see the tier table above).',
      },
      {
        q: 'Getting a 403 like "Host not allowed" or "Origin not allowed"?',
        a: 'That is your local service enforcing a Host/Origin allowlist: the tunnel forwards the visitor\'s Host and Origin (the public domain) unchanged. Note: pages built by Vite & co. load JS via <code>&lt;script type="module"&gt;</code>, and browsers always send an Origin header on those requests — the classic symptom is the HTML loads but every JS/CSS file 403s. Either allowlist your tunnel domain in the service settings, or start the tunnel with <code>--rewrite-host</code>, which rewrites Host to the target address (e.g. <code>127.0.0.1:3000</code>) and strips Origin/Referer: <code>soulink 3000 --rewrite-host</code>. Over the tunnel the page and its assets are same-origin, so neither header is needed.',
      },
      {
        q: 'Does it support WebSocket?',
        a: 'Yes. The tunnel carries a raw byte stream, so WebSocket upgrades pass through transparently with zero configuration.',
      },
      {
        q: 'Can I expose a service on another machine in my LAN?',
        a: 'Yes — registered users can point a tunnel at any internal address: <code>soulink http 192.168.1.10:8080</code>. For safety, guest tunnels only allow loopback targets.',
      },
      {
        q: 'Download fails or is slow?',
        a: "The installer falls back to built-in mirrors automatically when GitHub is unreachable, with sha256 verification throughout. You can also pin a mirror: <code>SOULINK_GITHUB_MIRROR=gh-proxy.com</code> (Windows: <code>$env:SOULINK_GITHUB_MIRROR='gh-proxy.com'</code>). On Windows you can also grab the zip from Releases.",
      },
      {
        q: 'What happens when I exceed my quota?',
        a: 'Tunnels are paused once the daily/monthly traffic limit is reached, and resume automatically in the next period. Live usage is visible in the console.',
      },
      {
        q: 'Is my traffic secure?',
        a: 'Traffic is encrypted end to end with HTTPS / wss. Note: TLS terminates at the hub, so the hub operator can technically see the plaintext being relayed — this is true of all relay-based tunneling services (ngrok included). The soulink hub is not open source at this time; evaluate sensitive use cases accordingly.',
      },
      {
        q: 'Does the domain stay valid?',
        a: 'Guest random domains are reclaimed after 4 hours; custom domains of registered users are durable. The client reconnects automatically with exponential backoff and tries to keep the original domain.',
      },
    ],
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
