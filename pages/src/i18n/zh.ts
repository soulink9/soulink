// 中文文案（默认语言，/ 路径）。命令与版本号留在组件中，这里只放 prose。
const zh = {
  locale: 'zh-CN',
  altPath: '/en/',
  altLabel: 'English',
  meta: {
    title: '闪灵速递 Soulink — 将你的 AI 闪光点，一键共享给朋友',
    desc: 'Soulink（闪灵速递）是开源的内网穿透工具：一条命令安装，一条命令把本地端口暴露为公网 HTTPS 地址。免注册匿名即用，注册可绑定固定域名。',
  },
  nav: { brand: '闪灵速递 Soulink', features: '特性', quickstart: '快速上手', tiers: '等级', faq: 'FAQ', github: 'GitHub', theme: '切换主题' },
  hero: {
    badge: '开源 · MIT · 免注册',
    titleA: '将你的 AI 闪光点，',
    titleB: '一键共享给朋友',
    sub: '无需注册、无需配置。安装后运行一条命令，本地运行的 AI 应用与服务立刻变成公网 HTTPS 地址，随手分享给朋友。',
    ctaStart: '快速上手',
    ctaGithub: 'GitHub',
    termComment1: '# 安装（macOS / Linux）',
    termComment1Win: '# 安装（Windows PowerShell）',
    termComment2: '# 把本地 3000 端口暴露到公网',
    termNote: '匿名即用 · 随机域名 · 有时效限制',
  },
  features: {
    title: '特性',
    items: [
      { t: '匿名即用', d: '不注册、不登录。装完直接 soulink 3000，随机域名立即可访问。' },
      { t: '固定子域名', d: '注册后绑定专属域名（如 myapp.apps.soulink.cam），重启不失效。' },
      { t: '单二进制', d: 'Go 编写、零依赖，macOS / Linux / Windows 全平台，一条命令安装。' },
      { t: '配额与限速', d: '账号级限流与日/月流量限额，匿名通道也有公平共享保护。' },
      { t: '站内信支持', d: '控制台内直接联系管理员，问题反馈不用翻墙提 issue。' },
      { t: '客户端开源', d: 'MIT 协议，客户端在 GitHub 完全开源，行为透明可审计。' },
    ],
  },
  quickstart: {
    title: '快速上手',
    s1t: '1. 安装',
    s1d: '一条命令安装：',
    s1note: '脚本会自动尝试镜像加速；下载慢也可显式指定镜像：',
    s2t: '2. 开隧道（匿名，免注册）',
    s2d: '把本地 3000 端口暴露到公网，自动分配随机域名：',
    s3t: '3. 绑定固定域名（可选）',
    s3d: '到 hub 注册账号并复制 token，配置后即可使用固定子域名：',
    winZip: '也可以从 Releases 下载 zip 解压即用。',
  },
  tiers: {
    title: '等级与配额',
    desc: '匿名通道开箱即用；注册账号解锁更高配额与固定域名。',
    cols: ['等级', '域名', '并发隧道', '限速', '流量限额'],
    rows: [
      ['guest（匿名）', '随机域名，4 小时回收', '全站共享 20 条', '256 KB/s 共享', '200 MB/日·条'],
      ['free', '自定义子域名', '3 条（随机 ≤ 2）', '512 KB/s', '100 MB/日 · 1 GB/月'],
      ['pro', '可预留 3 个固定域名', '10 条', '1 MB/s', '5 GB/日 · 100 GB/月'],
      ['max', '可预留 10 个固定域名', '50 条', '5 MB/s', '100 GB/日 · 1 TB/月'],
    ],
    note: '配额以 hub 实际配置为准，以上为本站（soulink.cam）当前默认值。',
  },
  faq: {
    title: '常见问题',
    items: [
      {
        q: 'soulink 和 ngrok、frp 有什么区别？',
        a: '开源（MIT）、免注册匿名即用：安装后一条命令就有公网 HTTPS 地址，不需要账号。注册后可绑定固定子域名。客户端是单二进制，macOS / Linux / Windows 全平台。',
      },
      {
        q: '匿名使用有什么限制？',
        a: 'guest 通道不注册不登录：随机域名（<code>xxx.apps.soulink.cam</code>）、4 小时回收、全站共享并发与限速、每条隧道日流量 200 MB。适合临时演示与调试；需要固定域名和更高配额就注册账号（见上方等级表）。',
      },
      {
        q: '访问时报「Host 不被允许（只允许回环地址）」之类的 403？',
        a: '这是你本地服务自己的 Host 头白名单：隧道把访客的 Host（公网域名）原样转发给了本地服务，而它只接受 localhost / 127.0.0.1。两种解法：在本地服务设置里放行你的隧道域名；或开隧道时加 <code>--rewrite-host</code>，把 Host 改写成目标地址（如 <code>127.0.0.1:3000</code>）再转发：<code>soulink 3000 --rewrite-host</code>。',
      },
      {
        q: '支持 WebSocket 吗？',
        a: '支持。隧道传输原始字节流，WebSocket 升级流量天然透明穿透，无需任何配置。',
      },
      {
        q: '能分享局域网里另一台机器的服务吗？',
        a: '可以。注册用户的隧道可指向任意内网地址：<code>soulink http 192.168.1.10:8080</code>。guest 通道出于安全只允许本机回环地址。',
      },
      {
        q: '下载安装失败或很慢？',
        a: "安装脚本内置镜像自动回落：直连 GitHub 失败会依次尝试内置镜像，全程 sha256 校验。也可显式指定镜像：<code>SOULINK_GITHUB_MIRROR=gh-proxy.com</code>（Windows：<code>$env:SOULINK_GITHUB_MIRROR='gh-proxy.com'</code>）。Windows 还可以直接从 Releases 下载 zip。",
      },
      {
        q: '超过配额会怎样？',
        a: '达到日/月流量限额后隧道暂停服务，次日/次月自动恢复。控制台可实时查看用量。',
      },
      {
        q: '我的流量安全吗？',
        a: '访客到你的服务全程 HTTPS / wss 加密。需要说明：TLS 在 hub 终止，hub 运营方技术上可见转发明文——这是所有中转式隧道服务的共同特性（ngrok 同）。soulink hub 暂不开源，敏感场景请谨慎评估。',
      },
      {
        q: '域名会一直有效吗？',
        a: 'guest 随机域名 4 小时回收；注册用户的自定义子域名长期有效。客户端断线自动指数退避重连，重连后尽量保持原子域名。',
      },
    ],
  },
  footer: {
    tagline: '开源内网穿透，一条命令上线。',
    links: '链接',
    github: 'GitHub',
    releases: 'Releases',
    console: '控制台',
    license: 'MIT License',
  },
};
export default zh;
