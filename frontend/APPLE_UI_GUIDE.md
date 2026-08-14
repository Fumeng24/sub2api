# Apple UI Refactor Guide

This project should read like a focused Apple-style product interface, adapted
for an API operations console. Apply this guide before making page-level changes.

## Visual Language

- Use quiet neutral surfaces: `#f5f5f7`, `#ffffff`, `#1d1d1f`, and deep black in dark mode.
- Use Apple blue only for primary actions and selected states: light `#0071e3`, dark `#2997ff`.
- Keep radius at `8px` for cards, dialogs, tables, and controls unless the component is a pill button or avatar.
- Avoid gradients, decorative blobs, nested cards, and loud shadows.
- Use real content density for admin/user tools: calm, scan-friendly, not marketing-heavy.
- Prefer one clear CTA per panel. Secondary actions should be subdued.
- Homepage hero copy should lead with user pain points and trust claims in this order: official capability, privacy, charge protection, stability.
- Homepage hero should use a concise promise headline and a shorter supporting line; avoid repeating the same claim in multiple adjacent blocks.
- Homepage visuals should stay restrained and high-contrast in both themes; dark mode must not rely on pure white text on bright surfaces.

## Typography

- Use the existing system font stack.
- Dashboard/page titles: `24-32px`, semibold, normal tracking.
- Card titles: `16-20px`, semibold.
- Table and form text: `13-14px`, compact but readable.
- Do not use negative letter spacing or viewport-scaled font sizes.

## Components

- `.btn`, `.input`, `.card`, `.table-*`, `.dropdown`, `.modal-*`, `.dialog-*`, `.sidebar-*`, `.page-*`
  are shared tokens in `src/style.css`. Reuse or improve those classes instead of inventing local one-off palettes.
- New user-facing copy must go through i18n. Do not add hard-coded Chinese or English in Vue templates unless the file is intentionally non-localized.
- Avoid implementation wording in user pages. For example, image generation should speak about
  "image access" / "生图访问能力" instead of internal key creation, unless the page is explicitly
  the API key management surface.
- Buttons should feel like iOS/macOS controls:
  - primary: blue pill or 8px rounded rectangular button, white text
  - secondary: white/black surface with thin border
  - ghost: transparent with subtle hover
- Tables should use white/dark elevated surfaces, sticky headers, low-contrast borders, and no heavy gridlines.
- Forms should use 8px controls, clear focus ring, and subdued helper text.
- Modals should use black translucent overlay, 8px panel, clean header/body/footer separation.

## Dark Mode

- Every changed surface must work under `.dark`.
- Use CSS variables from `src/style.css` when possible:
  `--apple-bg`, `--apple-surface`, `--apple-surface-elevated`, `--apple-text`,
  `--apple-muted`, `--apple-border`, `--apple-blue`, `--apple-blue-hover`.
- Do not leave white backgrounds or gray text without dark variants.

## Page Rules

- Operational pages should be dense, unframed layouts with repeated cards only for repeated items.
- Avoid card-in-card. If a panel contains a list/table/form, it is already the frame.
- Keep mobile layouts stable: buttons wrap, text truncates intentionally, table cards use clear label/value rows.
- Preserve the existing custom homepage rendering path (`home_content` as HTML or iframe URL). Apple-style defaults must not remove that admin-controlled capability.
- Keep homepage copy trust-first and concise: official capability, privacy boundary, charge protection, stability. Avoid onboarding/tutorial language, `/keys`, key creation, or step-by-step guidance on the public home screen.
- Do not add homepage `/keys` CTAs for authenticated users; send returning users to the dashboard or their existing workflow instead.
- API key management should read as a credential tool, not a landing page. Keep the header/toolbar compact, combine status, Base URL, refresh, create, and filters into one scan-friendly area, make the key value the primary row content with immediate copy affordance on mobile, and collapse low-frequency row actions behind a More menu.

## Verification

- Run `pnpm -C frontend exec eslint <changed files>` for touched Vue/TS files.
- Run `pnpm -C frontend typecheck` after shared component or broad page changes.
- Search changed frontend files for `rounded-2xl`, `rounded-3xl`, `shadow-xl`, `shadow-2xl`, `bg-gradient`, `blur-3xl`, `orb`, `bokeh`, and hard-coded user-facing text before handing off.
- For homepage edits, also check for repeated trust claims, oversized hero copy on mobile, and any surface that loses contrast in `.dark`.
- Prefer visual checks across light/dark and mobile/desktop when browser tooling is available.
