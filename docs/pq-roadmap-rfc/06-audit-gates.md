# 6) Audit-Gates als Pflichtbestandteil

## Zweck und Grundprinzip
Audit-Gates sind der verbindliche Sicherheits- und Qualitaetsmechanismus zwischen den Roadmap-Phasen. Sie sollen verhindern, dass nicht ausreichend gepruefte Annahmen aus Prototyp- oder Migrationsarbeit in den Live-Betrieb uebergehen.

Grundsatz ist: Ohne bestandenes Gate keine Empfehlung fuer den Uebergang in den naechsten risikorelevanten Schritt. Ausnahmen sind nur als explizite Governance-Risikoannahme zulaessig und muessen mit Begruendung, Rest-Risiko und Nachbesserungsplan dokumentiert werden.

## Audit-Logik ueber die Phasen
Das RFC verwendet unterschiedliche Gate-Typen je nach Phase:
- `Phase Omega`: Konsistenz- und Governance-Gate (inhaltlich/prozessual, kein Code-Security-Audit).
- `Phase A`: zwei verpflichtende technische Security-Gates (Prototyp und Migrationskomponenten) fuer den Konsenspfad.
- `Phase B`: ein verpflichtendes technisches Security-Gate nach Prototypbetrieb fuer den hybriden Wallet-/Tx-Pfad.
- `Phase C`: kein einzelnes starres "One-Shot"-Gate, sondern ein fortlaufendes Assurance-Modell fuer Produkt-, Betriebs- und Lieferkettenrisiken.

Damit bleibt die Gate-Intensitaet an der Kritikalitaet der jeweiligen Phase ausgerichtet: maximal strikt im Konsenspfad, strikt im Wallet-/Tx-Pfad, kontinuierlich-im-betrieblich im Client-Oekosystem.

## Gate-Matrix (verbindliche Mindestlogik)
1. `Omega-Gate`: Phase Omega muss abgeschlossen sein, bevor A-C starten.
2. `A-Gate 1` (nach A2): Audit des PQ-Konsens-Prototyps als Voraussetzung fuer A4.
3. `A-Gate 2` (nach A4): Audit der Konsens-Migrationskomponenten als Voraussetzung fuer produktiven Migrationsentscheid.
4. `B-Gate` (nach B2): Audit des hybriden Wallet-/Tx-Pfads als Voraussetzung fuer B4.
5. `B4` hat bewusst kein zweites verpflichtendes separates Gate; bei hoher Komplexitaet kann Governance zusaetzliches Re-Audit verlangen.
6. `C-Assurance`: regelmaessige Pruefzyklen statt einmaligem Freigabe-Gate.

## Pflichtbestandteile jedes Audit-Pakets
Jedes Gate muss mindestens folgende Artefakte enthalten:
- Audit-Ziele und expliziter Scope (inklusive Nicht-Scope).
- Audit-Typ (intern, extern oder kombiniert) mit Rollenaufteilung.
- Bereitgestellte Artefakte fuer Auditoren (Code, Architektur, Testnachweise, Threat Model, Betriebsdokumente).
- Findings nach Kritikalitaetsklasse.
- Re-Audit-Regeln je Kritikalitaetsklasse.
- Formale Go/No-Go-Empfehlung mit Rest-Risiko-Bewertung.

## Findings-Klassen und Blocker-Regeln
Verbindliche Mindestklassen:
- `Critical`: direkter Blocker, kein Uebergang ohne Fix und Re-Audit.
- `High`: Blocker fuer produktionsnahe oder produktive Schritte; Ausnahme nur per expliziter Governance-Risikoannahme.
- `Medium`: kein automatischer Blocker, aber mit Frist und Trackingpflicht.
- `Low/Info`: dokumentations- und backlogpflichtig, kein Gate-Blocker.

Ein Gate gilt nur dann als bestanden, wenn alle `Critical` Findings geschlossen sind und keine ungesteuerte `High`-Risikoexposition verbleibt.

## Re-Audit- und Change-Regeln
Re-Audit ist verpflichtend bei:
- Fixes fuer `Critical` oder `High` Findings.
- Aenderungen an kryptografischen Kernpfaden (Signatur- oder Verifikationslogik).
- Aenderungen an Cutover-, Migrations- oder Key-Binding-Mechaniken.
- Aenderungen mit Einfluss auf Determinismus, Safety/Liveness oder Key-Management.

Nach bestandenem Gate sind relevante Code- und Konfigurationsaenderungen bis zum naechsten Meilenstein auditzugbezogen zu dokumentieren, um Scope-Drift zu vermeiden.

## Pflichtpruefpunkte im Konsenspfad (Phase A)
- Safety/Liveness-Eigenschaften des geforkten CometBFT unter PQ-Integration.
- Korrekte Verifikation in Proposal-, Vote- und Commit-Pfaden.
- Determinismus ueber Upgrade- und Cutover-Fenster.
- DoS-Risiken durch groessere Signaturen, veraenderte Verifikationskosten und Lastspitzen.
- Hardening von PrivValidator-/Remote-Signer-Schnittstellen.
- Korrektheit und Manipulationsresistenz des Key-Binding (`klassisch -> PQ`) sowie Snapshot-/Aktivierungslogik.
- Negative Tests (malformed Inputs, Grenzgroessen, Mixed-Version-Szenarien).

## Pflichtpruefpunkte im Wallet-/Tx-Pfad (Phase B)
- Korrekte SigVerification und Ante-Handler-Logik fuer hybride Signaturpfade.
- Replay-, Malleability- und Domain-Separation-Pruefungen.
- Gas-/Fee-Haerte und Anti-Spam/DoS-Verhalten im hybriden Betrieb.
- Korrektheit von Codec, Serialisierung und relevanten Ableitungsregeln.
- Key-Management- und Recovery-Flows fuer Nutzer und Integratoren.
- Kompatibilitaetstests im Einfuehrungs- und Migrationsfenster.

## Pflichtpruefpunkte im Client-/Betriebskontext (Phase C)
Phase C fokussiert den dauerhaften Betrieb eines Terra-Classic-eigenen Wallet-/Explorer-Stacks und der Traegerstruktur. Dafuer gelten kontinuierliche Assurance-Pflichten:
- Betriebs- und Hosting-Sicherheit (Web, Backend, Node-Betrieb, Monitoring).
- Lieferkettensicherheit und Release-Integritaet (Build, Distribution, Signierung, Publisher-Accounts).
- Governance- und Eigentuemerschaftskontrolle kritischer Assets (DNS, Repositories, Paket-/App-Distribution).
- Incident-, Patch- und Vulnerability-Response-Prozesse mit nachvollziehbaren Eskalationswegen.
- Externes PQ-Standards-Monitoring und Rueckkopplung in technische Entscheidungen.

## Dokumentationspflicht je Gate
Jedes Gate erzeugt einen auditierbaren Entscheidungsdatensatz:
- Scope und Version der geprueften Artefakte.
- Findings-Liste mit Status und Verantwortlichkeit.
- Risikoakzeptanzen (falls vorhanden) inkl. Entscheidungsinstanz.
- Go/No-Go-Entscheidung und freigegebener naechster Schritt.

Damit wird sichergestellt, dass die Roadmap nicht nur technisch umgesetzt, sondern entlang ihrer risikokritischen Uebergaenge nachvollziehbar und reproduzierbar gesteuert wird.
