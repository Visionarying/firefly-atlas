"""把 docs/ROADMAP.md 渲染成可离线阅读的 docs/ROADMAP.html。

用法（在仓库根目录执行）：
    python scripts/build-docs.py

改了 ROADMAP.md 之后重新跑一次即可，HTML 由脚本生成，不要手改。
"""

import io
import re
from pathlib import Path

import markdown

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "docs" / "ROADMAP.md"
OUT = ROOT / "docs" / "ROADMAP.html"

CSS = """
  :root {
    --fg: #1f2328;
    --fg-muted: #59636e;
    --border: #d1d9e0;
    --bg-soft: #f6f8fa;
    --accent: #0f6e56;
    --accent-soft: #e8f5f0;
  }
  * { box-sizing: border-box; }
  html { scroll-behavior: smooth; }
  body {
    margin: 0;
    padding: 48px 24px 96px;
    background: #ffffff;
    color: var(--fg);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Microsoft YaHei", "PingFang SC", sans-serif;
    font-size: 15px;
    line-height: 1.75;
    -webkit-font-smoothing: antialiased;
  }
  .wrap { max-width: 880px; margin: 0 auto; }
  h1 { font-size: 30px; font-weight: 600; line-height: 1.3; margin: 0 0 8px; letter-spacing: -0.01em; }
  h2 { font-size: 21px; font-weight: 600; margin: 48px 0 16px; padding-bottom: 8px; border-bottom: 1px solid var(--border); scroll-margin-top: 24px; }
  h3 { font-size: 17px; font-weight: 600; margin: 28px 0 10px; }
  p { margin: 12px 0; }
  a { color: var(--accent); text-decoration: none; border-bottom: 1px solid var(--accent-soft); }
  a:hover { border-bottom-color: var(--accent); }
  blockquote {
    margin: 16px 0;
    padding: 12px 16px;
    background: var(--bg-soft);
    border-left: 3px solid var(--accent);
    border-radius: 0 6px 6px 0;
    color: var(--fg-muted);
  }
  blockquote p { margin: 4px 0; }
  .toc {
    margin: 24px 0 8px;
    padding: 16px 20px;
    background: var(--bg-soft);
    border: 1px solid var(--border);
    border-radius: 8px;
    display: flex;
    flex-wrap: wrap;
    gap: 6px 18px;
    align-items: baseline;
  }
  .toc-title { font-size: 13px; font-weight: 600; color: var(--fg-muted); margin-right: 4px; }
  .toc a { font-size: 13.5px; border-bottom: none; }
  .toc a:hover { text-decoration: underline; }
  table { width: 100%; border-collapse: collapse; margin: 16px 0; font-size: 14px; }
  th, td { border: 1px solid var(--border); padding: 9px 12px; text-align: left; vertical-align: top; }
  th { background: var(--bg-soft); font-weight: 600; }
  tr:nth-child(even) td { background: #fcfcfd; }
  code {
    font-family: "Cascadia Code", Consolas, "Courier New", monospace;
    font-size: 13px;
    background: var(--bg-soft);
    padding: 2px 5px;
    border-radius: 4px;
  }
  pre {
    background: var(--bg-soft);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 14px 16px;
    overflow-x: auto;
    line-height: 1.6;
  }
  pre code { background: none; padding: 0; font-size: 13px; }
  ul, ol { padding-left: 24px; margin: 12px 0; }
  li { margin: 5px 0; }
  hr { border: none; border-top: 1px solid var(--border); margin: 40px 0; }
  strong { font-weight: 600; }
  em { color: var(--fg-muted); }
  @media (max-width: 640px) {
    body { padding: 28px 16px 64px; font-size: 14px; }
    h1 { font-size: 24px; }
    h2 { font-size: 19px; }
    table { font-size: 13px; }
    th, td { padding: 7px 9px; }
    .toc { gap: 4px 12px; }
  }
  @media print {
    body { padding: 0; }
    .toc { display: none; }
    h2 { page-break-after: avoid; }
    table, pre { page-break-inside: avoid; }
  }
"""

TEMPLATE = """<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Firefly Panel 版本规划与功能取舍方案</title>
<style>__CSS__</style>
</head>
<body>
<div class="wrap">
__BODY__
</div>
</body>
</html>
"""


def build_toc(body: str) -> str:
    heads = re.findall(r'<h2 id="([^"]*)">(.*?)</h2>', body)
    if not heads:
        return ""
    links = []
    for hid, htext in heads:
        plain = re.sub(r"<[^>]+>", "", htext)
        links.append(f'<a href="#{hid}">{plain}</a>')
    return '<nav class="toc"><span class="toc-title">目录</span>' + "".join(links) + "</nav>\n"


def main() -> None:
    if not SRC.exists():
        raise SystemExit(f"找不到源文件: {SRC}")

    text = SRC.read_text(encoding="utf-8")
    body = markdown.markdown(
        text,
        extensions=["tables", "fenced_code", "toc", "sane_lists"],
        extension_configs={"toc": {"permalink": False}},
    )

    nav = build_toc(body)
    idx = body.find("<h2 id=")
    body = nav + body if idx < 0 else body[:idx] + nav + body[idx:]

    html = TEMPLATE.replace("__CSS__", CSS).replace("__BODY__", body)
    OUT.write_text(html, encoding="utf-8")
    print(f"已生成 {OUT.relative_to(ROOT)}，{len(html)} 字节，表格 {html.count('<table>')} 个")


if __name__ == "__main__":
    main()
