.PHONY: all install user-install user-uninstall devel-install devel-uninstall clean watch build

all: build

build:
	npm install
	npm run build

watch:
	npm run watch

install:
	mkdir -p $(DESTDIR)/usr/local/share/cockpit/comquad
	cp -r dist/* $(DESTDIR)/usr/local/share/cockpit/comquad/
	cp src/manifest.json $(DESTDIR)/usr/local/share/cockpit/comquad/

user-install:
	mkdir -p ~/.local/share/cockpit/comquad
	cp -r dist/* ~/.local/share/cockpit/comquad/
	cp src/manifest.json ~/.local/share/cockpit/comquad/

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
