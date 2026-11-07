"""一次改完所有版本号，避免发版时漏改其中某一处。

用法（在仓库根目录执行）：
    python scripts/bump-version.py 1.3.2

会同步这几处：
    service/global/global.go              Version = "x.y.z"
    web/package.json                      "version": "x.y.z"
    web/src/components/SettingsModal.vue  const version = ref('x.y.z') 与关于页标签
    Dockerfile                            ARG VERSION=x.y.z
    docker-compose.yml                    VERSION: "x.y.z" 与 image: firefly-atlas:x.y.z
    README.md                             标题、版本命名说明与 ghcr.io 拉取地址

CHANGELOG 与 ROADMAP 里的历史版本号不会被改动。
"""

import io
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# (文件, 正则, 替换模板, 说明)  —— 模板里 {v} 代表新版本号
# 一律用 \g<1> 而不是 \1：版本号以数字开头，\1 会和后面的数字连成 \11 被当成第 11 组。
RULES = [
    ("service/global/global.go", r'(Version\s*=\s*)"[^"]*"', r'\g<1>"{v}"', "后端默认版本"),
    ("web/package.json", r'("version"\s*:\s*)"[^"]*"', r'\g<1>"{v}"', "前端包版本"),
    ("web/src/components/SettingsModal.vue", r"(const version = ref\(')[^']*('\))", r"\g<1>{v}\g<2>", "关于页版本号"),
    ("web/src/components/SettingsModal.vue", r"(Version \{\{ version \}\} · V)\d+\.\d+\.\d+", r"\g<1>{v}", "关于页版本标签"),
    ("Dockerfile", r"(ARG VERSION=)\d+\.\d+\.\d+", r"\g<1>{v}", "镜像构建默认版本"),
    ("docker-compose.yml", r'(VERSION:\s*)"[^"]*"', r'\g<1>"{v}"', "compose 构建参数"),
    ("docker-compose.yml", r"(image:\s*firefly-atlas:)[^\s]+", r"\g<1>{v}", "compose 镜像标签"),
    ("README.md", r"(?m)^(# Firefly Atlas v)\d+\.\d+\.\d+", r"\g<1>{v}", "README 标题"),
    ("README.md", r"(版本命名：`v)\d+\.\d+\.\d+(` 代号)", r"\g<1>{v}\g<2>", "README 版本命名说明"),
    ("README.md", r"(firefly-atlas:)\d+\.\d+\.\d+", r"\g<1>{v}", "文档里的拉取地址"),
]


def main() -> None:
    if len(sys.argv) != 2 or not re.fullmatch(r"\d+\.\d+\.\d+", sys.argv[1]):
        print("用法: python scripts/bump-version.py 1.3.2")
        raise SystemExit(2)

    version = sys.argv[1]
    print(f"目标版本: {version}\n")

    changed_files: set[str] = set()
    skipped = 0
    for rel, pattern, template, label in RULES:
        path = ROOT / rel
        if not path.exists():
            print(f"  跳过 {rel}：文件不存在")
            continue

        raw = path.read_bytes()
        text = raw.decode("utf-8")
        new_text, count = re.subn(pattern, template.replace("{v}", version), text)

        if count == 0:
            print(f"  未匹配 {rel}：{label}（正则没有命中，检查文件是否被改过）")
            skipped += 1
            continue

        if new_text == text:
            print(f"  已是 {version}：{rel} · {label}")
            continue

        path.write_bytes(new_text.encode("utf-8"))
        changed_files.add(rel)
        print(f"  已更新 {rel} · {label}（{count} 处）")

    print(f"\n完成：改动 {len(changed_files)} 个文件，{skipped} 处未匹配。")
    if skipped:
        print("有未匹配项，请人工核对后再提交。")
        raise SystemExit(1)

    print("\n下一步建议：")
    print("  1. 更新 CHANGELOG.md 与本版说明")
    print("  2. 运行 go vet ./... 与前端构建")
    print(f"  3. 提交并打标签 v{version}")


if __name__ == "__main__":
    main()
