.PHONY: all install user-install user-uninstall devel-install devel-uninstall clean watch build po-pot po-update po-build test

all: build

build: po-build
	npm install
	npm run build

watch:
	npm run watch

install:
	rm -rf $(DESTDIR)/usr/local/share/cockpit/comquad
	mkdir -p $(DESTDIR)/usr/local/share/cockpit/comquad
	cp -r dist/* $(DESTDIR)/usr/local/share/cockpit/comquad/

user-install:
	rm -rf ~/.local/share/cockpit/comquad
	mkdir -p ~/.local/share/cockpit/comquad
	cp -r dist/* ~/.local/share/cockpit/comquad/

user-uninstall:
	rm -rf ~/.local/share/cockpit/comquad

devel-install: build
	mkdir -p ~/.local/share/cockpit
	ln -sf $$(pwd)/dist ~/.local/share/cockpit/comquad

devel-uninstall:
	rm -f ~/.local/share/cockpit/comquad

clean:
	rm -rf dist node_modules

eslint:
	npm run eslint

stylelint:
	npm run stylelint

codecheck: eslint stylelint

test:
	npm test

po-pot:
	@echo "Extracting strings to po/comquad.pot..."
	@mkdir -p po
	@# Extract strings from source files using grep-based extraction
	@# This is a simplified extraction; for production use xgettext or babel-xgettext
	@grep -rohP '_\("([^"]+)"\)' src/ | sed 's/_(\|"//g' | sort -u | \
		awk 'BEGIN { print "# SOME DESCRIPTIVE TITLE." ; \
		           print "# Copyright (C) YEAR THE PACKAGE'\''S COPYRIGHT HOLDER" ; \
		           print "# This file is distributed under the same license as the cockpit-comquad package." ; \
		           print "#" ; \
		           print "#, fuzzy" ; \
		           print "msgid \"\"" ; \
		           print "msgstr \"\"" ; \
		           print "\"Project-Id-Version: cockpit-comquad\\n\"" ; \
		           print "\"Report-Msgid-Bugs-To: \\n\"" ; \
		           print "\"POT-Creation-Date: " $$(date -u +"%Y-%m-%d %H:%M+0000") "\\n\"" ; \
		           print "\"PO-Revision-Date: YEAR-MO-DA HO:MI+ZONE\\n\"" ; \
		           print "\"Last-Translator: FULL NAME <EMAIL@ADDRESS>\\n\"" ; \
		           print "\"Language-Team: LANGUAGE <LL@li.org>\\n\"" ; \
		           print "\"Language: \\n\"" ; \
		           print "\"MIME-Version: 1.0\\n\"" ; \
		           print "\"Content-Type: text/plain; charset=UTF-8\\n\"" ; \
		           print "\"Content-Transfer-Encoding: 8bit\\n\"" ; \
		           print "" } \
		     { print "#: src/" ; print "msgid \"" $$0 "\"" ; print "msgstr \"\"" ; print "" }' \
		> po/comquad.pot
	@echo "Generated po/comquad.pot"

po-update: po-pot
	@for lang in $$(cat po/LINGUAS 2>/dev/null | grep -v '^#'); do \
		if [ -f "po/$$lang.po" ]; then \
			echo "Updating po/$$lang.po..."; \
			msgmerge -U "po/$$lang.po" po/comquad.pot 2>/dev/null || true; \
		else \
			echo "Creating po/$$lang.po..."; \
			msginit -i po/comquad.pot -o "po/$$lang.po" -l $$lang --no-translator 2>/dev/null || \
			cp po/comquad.pot "po/$$lang.po"; \
		fi; \
	done

po-build:
	@mkdir -p dist/po
	@for lang in $$(cat po/LINGUAS 2>/dev/null | grep -v '^#'); do \
		if [ -f "po/$$lang.po" ]; then \
			echo "Compiling po/$$lang.po to dist/po.$$lang.js..."; \
			python3 po/compile.py "po/$$lang.po" "dist/po.$$lang.js"; \
		fi; \
	done

test-browser-image:
	podman build -t cockpit-comquad-test:latest -f test/browser/Containerfile ../..

test-browser: test-browser-image build
	podman run --rm --privileged \
		--cgroupns=host \
		-v /sys/fs/cgroup:/sys/fs/cgroup:rw \
		-v $(shell cd .. && pwd):/workspace:z \
		-w /workspace/cockpit-comquad \
		cockpit-comquad-test:latest \
		/run
