---
name: VM Automation Console
colors:
  surface: '#fcf9f8'
  surface-dim: '#dcd9d9'
  surface-bright: '#fcf9f8'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f6f3f2'
  surface-container: '#f0eded'
  surface-container-high: '#eae7e7'
  surface-container-highest: '#e5e2e1'
  on-surface: '#1b1b1c'
  on-surface-variant: '#59413d'
  inverse-surface: '#303030'
  inverse-on-surface: '#f3f0ef'
  outline: '#8d706c'
  outline-variant: '#e1bfb9'
  surface-tint: '#b02d21'
  primary: '#9e2016'
  on-primary: '#ffffff'
  primary-container: '#c0392b'
  on-primary-container: '#ffe5e1'
  inverse-primary: '#ffb4a9'
  secondary: '#006d37'
  on-secondary: '#ffffff'
  secondary-container: '#7bf8a1'
  on-secondary-container: '#007239'
  tertiary: '#005875'
  on-tertiary: '#ffffff'
  tertiary-container: '#007296'
  on-tertiary-container: '#d3eeff'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#ffdad5'
  primary-fixed-dim: '#ffb4a9'
  on-primary-fixed: '#410000'
  on-primary-fixed-variant: '#8e130c'
  secondary-fixed: '#7efba4'
  secondary-fixed-dim: '#61de8a'
  on-secondary-fixed: '#00210c'
  on-secondary-fixed-variant: '#005228'
  tertiary-fixed: '#c0e8ff'
  tertiary-fixed-dim: '#80d0f8'
  on-tertiary-fixed: '#001e2b'
  on-tertiary-fixed-variant: '#004d66'
  background: '#fcf9f8'
  on-background: '#1b1b1c'
  surface-variant: '#e5e2e1'
typography:
  headline-lg:
    fontFamily: Inter
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 24px
  headline-md:
    fontFamily: Inter
    fontSize: 15px
    fontWeight: '600'
    lineHeight: 20px
  headline-sm:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '600'
    lineHeight: 16px
  body-lg:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
  body-md:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 16px
  body-sm:
    fontFamily: Inter
    fontSize: 11px
    fontWeight: '400'
    lineHeight: 14px
  label-lg:
    fontFamily: JetBrains Mono
    fontSize: 12px
    fontWeight: '500'
    lineHeight: 16px
  label-md:
    fontFamily: JetBrains Mono
    fontSize: 11px
    fontWeight: '400'
    lineHeight: 14px
  label-sm:
    fontFamily: JetBrains Mono
    fontSize: 10px
    fontWeight: '400'
    lineHeight: 12px
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  gutter: 0.5rem
  margin: 0.5rem
  space-xs: 0.125rem
  space-sm: 0.25rem
  space-md: 0.5rem
  space-lg: 0.75rem
  space-xl: 1rem
---

## Brand & Style

This design system targets systems engineers, automation developers, and lab administrators managing VirtualBox instances. The aesthetic rejects decorative modern web conventions—such as heavy blur filters, floating elevation layers, and oversized touch targets—in favor of the disciplined, high-density ergonomics of native desktop workstation suites (Qt, Cocoa, classic systems management tooling).

The visual personality is clinical, reliable, and strictly utilitarian. Information architecture takes precedence over decorative white space: screen estate is optimized for throughput, state scanning, and rapid keyboard or pointer manipulation. Visual feedback relies on precise 1px geometric borders, strict state indicators, and acute semantic color contrasts rather than lighting metaphors.

## Colors

The palette establishes an authentic desktop application environment using flat, high-contrast structural values without gradients:

- **Canvas & Surface:**  forms the foundational application canvas, window frame, and tool dock background. Pure white  is reserved strictly for operational surfaces: VM data tables, inspector panels, configuration cards, and log consoles.
- **Structural Lines:** Base borders and separators use  (1px solid). Secondary sub-dividers use . Interactive hover states elevate borders to .
- **Text & Typography:** High-contrast neutral  provides clear legibility for primary labels and technical values. Secondary metadata, keyboard shortcut cues, and muted attributes use . Disabled text sits at .
- **Functional Semantics:** 
  - Primary Accent & Destructive Warning:  (active tabs, destructive actions, focus rings, and hover accent borders). An alternate  serves for urgent system failure alerts.
  - Active & Operational:  designates running VMs, active listening sockets, and successful automation runs.
  - Paused / Pending:  denotes paused VMs or pending snapshots.
  - Inactive / Powered Off:  indicates powered down or unallocated instances.

## Typography

Typography prioritizes density, tabular precision, and immediate legibility. Compact line heights prevent vertical bloat across data-heavy tables and property inspectors.

- **Primary Interface:**  handles all window titles, inspector headers, control buttons, and table headers. Metrics use tightened letter spacing ( on sizes 13px and above) to mimic native desktop text rasterization.
- **Data & Telemetry:**  handles IP addresses, MAC addresses, hex offsets, port configurations, serial port streams, automation scripts, and file system paths. Tabular figures are enforced across all numeric representations.

## Layout & Spacing

The layout model mimics an anchored desktop software window rather than a scrolling document. 

- **Structure:** A fixed top-level application shell composed of an edge-to-edge system titlebar/tabstrip, a 32px global utility toolbar, a collapsible left master pane (VM list/group inventory), a central workspace (console viewer/dashboard tabs), and an optional collapsible right properties inspector.
- **Rhythm:** Spacing follows a tight 4px and 8px base rhythm. Outer margins are minimal (0.5rem / 8px), maximizing viewport space for virtual machine frames and tabular telemetries. Panels are separated by explicit 1px splitter handles rather than margins.
- **Resizing:** Layout splitters allow user-dragged resizing between columns. Panes do not wrap; horizontal overflows inside log consoles or data tables introduce synchronous horizontal scrollbars.

## Elevation & Depth

This system avoids drop shadows, tinted ambient ambient blurs, and pseudo-skeuomorphic gradient fills. Visual layering is achieved exclusively through 2D structural planes and edge definitions:

- **Planes:** Base application background  sits at the deepest level. Working containers, table viewports, and active editor docks sit flush as  panels directly cut into the background via a 1px solid  perimeter.
- **Focus and Active State:** Depth is communicated by high-contrast border definition: an unselected frame carries a  boundary, an active or focused frame highlights with a 1px  line, and disabled regions flatten into  lines.
- **Overlays & Modals:** Context menus and essential dialog boxes use a clean 1px solid  or  perimeter with zero shadow, backed by  fill to stand distinctly against split panes.

## Shapes

Shapes across the system are crisp and mechanical, referencing classic Qt desktop widgets:

- **Base Radius:** All buttons, inputs, segmented controls, and inline status badges employ a razor-sharp 3px radius (token tier ).
- **Containers & Panels:** Top-level viewports, split panes, and data display cards use sharp, unrounded corners () where they meet viewport edges, or 3px when floating as independent inspector groups.
- **No Pill Radii:** Capsule or pill shapes are prohibited. Status indicators and chips remain strictly rectilinear with 2px to 3px maximum radii.

## Components

### Buttons & Utility Controls
- **Height & Sizing:** Standard utility buttons are strictly 26px tall, using  typography. Icon-only tool buttons are 26x26px.
- **Default Appearance:** Flat  surface, 1px solid  border,  text. 
- **States:** Hover triggers an accent border () and a subtle surface shift to . Active/pressed shifts background to . Focused elements feature a 1px solid  outline with 0px offset.
- **Primary / Destructive:** Primary operations utilize a solid  fill with  text. Destructive actions use the same treatment paired with clear confirmation tooltips.

### Desktop Tab Bar
- **Geometry:** Horizontal bar seated at 30px height, flush with the top of panels.
- **Tabs:** Inactive tabs rest on  with  text, separated by 1px vertical  rules. Active tabs display  background,  font, and a crisp 2px solid bottom underline in .

### Data Tables & VM Lists
- **Rows:** Dense 24px to 28px row height. Alternating rows can feature  striping.
- **Headers:** 24px height,  background, font  (600 weight), uppercase with 1px bottom border in .
- **Selection:** Selected row displays an unbroken  left 2px border indicator, paired with a solid  background across the row.

### Form Inputs & Checkboxes
- **Inputs:** 26px height, 1px solid  border,  background, 6px horizontal padding. Monospace font is invoked automatically for port, IP, path, and MAC fields.
- **Checkboxes & Radios:** Sharp 14x14px boxes with 2px corner radius. Unchecked shows  fill with 1px  border; checked shows  fill with a crisp white tick mark.

### Status Indicators & Chips
- **Status Dots:** 6px diameter square or circular glyphs with 0px blur. Green () indicates running, amber () indicates paused, and muted grey () indicates powered off.
- **Badges:** Compact tags with 2px internal padding, 1px solid border matching the status color, and monospace  text.

### Icons
- **Specification:** Crisp 16x16px vector glyphs with a continuous 1.5px stroke weight (Lucide standard). Icons must align cleanly to the pixel grid to prevent anti-aliasing blur.
