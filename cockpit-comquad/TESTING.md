# cockpit-comquad Testing and i18n Implementation

## Summary

This document describes the testing infrastructure and internationalization (i18n) support that has been added to cockpit-comquad.

## Internationalization (i18n)

### Implementation

All user-facing strings have been wrapped with the `_()` function for translation support:

- **src/i18n.ts**: Wrapper functions for `cockpit.gettext()`, `cockpit.ngettext()`, and `cockpit.format()`
- **All components**: Updated to use `_()` for translatable strings
- **po/comquad.pot**: Template file with 82 translatable strings
- **po/LINGUAS**: Configuration file for supported languages (currently empty)

### Build Integration

- **build.js**: Compiles `.po` files to JSON format in `dist/po/`
- **Makefile**: Added `po-build`, `po-pot`, and `po-update` targets
- **src/index.tsx**: Loads translations at startup based on browser language

### Usage

To add translations:
1. Copy `po/comquad.pot` to `po/<lang>.po` (e.g., `po/es.po`)
2. Translate the strings
3. Add the language code to `po/LINGUAS`
4. Run `make po-build` to compile translations

## Testing Infrastructure

### Tier 1: Unit Tests (vitest + React Testing Library)

**Configuration:**
- **vitest.config.ts**: Test configuration with jsdom environment
- **test/setup.ts**: Test setup with cockpit API mocks

**Test Coverage (60 tests):**
- **client.test.ts** (7 tests): API client functions
- **ErrorBoundary.test.tsx** (4 tests): Error boundary behavior
- **ProjectsList.test.tsx** (6 tests): Project list rendering
- **App.test.tsx** (10 tests): Main app component
- **StackDeploy.test.tsx** (7 tests): Stack deployment UI
- **DirectoryPicker.test.tsx** (11 tests): Directory selection
- **ProjectDetail.test.tsx** (15 tests): Project detail view

**Running Tests:**
```bash
npm test              # Run all tests
npm run test:watch    # Watch mode
```

### Tier 2: Browser Integration Tests (Selenium + Chromium)

**Infrastructure:**
- **test/browser/Containerfile**: Test container with cockpit, comquad, and browser
- **test/browser/vm.install**: Installation script for test environment
- **test/browser/run**: Test runner script
- **test/browser/test-comquad.py**: Browser automation tests

**Test Coverage:**
- **TestStackDashboard**: Stack list page tests
  - Page loads correctly
  - Projects are listed
  - Status badges display
  - Refresh button works
  - Deploy button present

- **TestProjectDetail**: Project detail page tests
  - Detail view loads
  - Services tab shows services
  - Containers tab shows containers
  - Resources tab shows resources
  - Action buttons present
  - Back button works

**Running Browser Tests:**
```bash
# Build test container
make test-image-cockpit

# Run browser tests
make integration-cockpit
```

## Makefile Targets

### cockpit-comquad/Makefile

- `make test`: Run unit tests
- `make po-build`: Compile translations
- `make po-pot`: Extract strings to .pot file
- `make po-update`: Update .po files from .pot
- `make test-browser-image`: Build browser test container
- `make test-browser`: Run browser tests

### Root Makefile

- `make test-image-cockpit`: Build cockpit test container
- `make integration-cockpit`: Run cockpit browser tests

## Files Modified/Created

### New Files
- `src/i18n.ts`
- `vitest.config.ts`
- `test/setup.ts`
- `test/*.test.tsx` (7 test files)
- `po/POTFILES.in`
- `po/LINGUAS`
- `po/comquad.pot`
- `test/browser/Containerfile`
- `test/browser/vm.install`
- `test/browser/run`
- `test/browser/test-comquad.py`

### Modified Files
- `package.json`: Added test dependencies and scripts
- `tsconfig.json`: Added test directory and vitest types
- `build.js`: Added po file compilation
- `Makefile`: Added test and i18n targets
- `src/index.tsx`: Added translation loading
- `src/cockpit.d.ts`: Added ngettext and format types
- All component files: Wrapped strings with `_()`

## Test Results

```
Test Files  7 passed (7)
Tests       60 passed (60)
Duration    ~3s
```

All unit tests pass successfully. Browser integration tests require the test container to be built and can be run with `make integration-cockpit`.
