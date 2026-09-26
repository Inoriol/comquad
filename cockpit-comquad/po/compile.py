#!/usr/bin/env python3
import re
import sys
import os
import json

po_file = sys.argv[1]
js_file = sys.argv[2]

content = open(po_file, 'r', encoding='utf-8').read()

language = ""
plural_forms = ""

header_match = re.search(r'^msgid\s+""\s*\n((?:msgstr\s+""\s*\n)?(?:"[^"]*"\s*\n)+)', content, re.M)
if header_match:
    header_lines = header_match.group(1)
    for line in header_lines.split('\n'):
        line_match = re.match(r'^"(.*)"$', line.strip())
        if line_match:
            line_content = line_match.group(1).replace('\\n', '\n')
            lang_match = re.match(r'Language:\s*(\S+)', line_content)
            if lang_match:
                language = lang_match.group(1)
            plural_match = re.match(r'Plural-Forms:\s*(.*)', line_content)
            if plural_match:
                plural_forms = plural_match.group(1)

RTL_LANGS = {"ar", "fa", "he", "ur"}
direction = "rtl" if language in RTL_LANGS else "ltr"

plural_expr = plural_forms
match = re.search(r'nplurals=[1-9];\s*plural=([^;]*);?$', plural_forms)
if match:
    plural_expr = f"(n) => {match.group(1)}"

chunks = ['{\n']
chunks.append(' "": {\n')
chunks.append(f'  "plural-forms": {plural_expr},\n')
chunks.append(f'  "language": "{language}",\n')
chunks.append(f'  "language-direction": "{direction}"\n')
chunks.append(' }')

entries = re.split(r'\n\n+', content)
for entry in entries:
    msgid_match = re.search(r'^msgid\s+"(.*)"', entry, re.M)
    msgstr_matches = re.findall(r'^msgstr(?:\[(\d+)\])?\s+"(.*)"', entry, re.M)

    if msgid_match and msgstr_matches:
        msgid = msgid_match.group(1)
        if not msgid:
            continue

        translations = [None]
        for idx, msgstr in msgstr_matches:
            translations.append(msgstr)

        key = json.dumps(msgid)
        chunks.append(f',\n {key}: [\n  null')
        for t in translations[1:]:
            chunks.append(f',\n  {json.dumps(t, ensure_ascii=False)}')
        chunks.append('\n ]')

chunks.append('\n}')

output = f"cockpit.locale({''.join(chunks)});\n"

os.makedirs(os.path.dirname(js_file), exist_ok=True)
with open(js_file, 'w', encoding='utf-8') as f:
    f.write(output)
