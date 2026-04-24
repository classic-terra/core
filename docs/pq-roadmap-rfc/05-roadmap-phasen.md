# 5) Roadmap-Phasen (Richtung, nicht Delivery-Plan)

## Phase Omega: RFC-Freeze und Startfreigabe (vorgelagert)
Phase Omega ist der vorgelagerte Abschluss der RFC-Entwurfsarbeit vor jeder technischen Umsetzung. Sie ist kein Delivery-Plan und keine Implementierungsphase, sondern der formale Uebergang von "Diskussion und Varianten" zu "verbindlicher Umsetzungsbasis". Ohne abgeschlossenes Omega-Gate beginnen die Umsetzungsphasen A bis C nicht.

Das Ziel der Phase ist eine belastbare inhaltliche Baseline fuer alle nachgelagerten technischen Entscheidungen. Dazu werden offene Konflikte nicht implizit stehen gelassen, sondern explizit entschieden, vertagt oder verworfen. Gleichzeitig wird festgelegt, welche Teile des RFC eingefroren sind und unter welchen Bedingungen Aenderungen nach Freeze zulaessig bleiben.

### Entwicklungspaket O1: Konsolidierung und Entscheidungsabschluss
O1 schliesst den RFC-Feedbackzyklus formal ab. Kommentare und Gegenpositionen aus Stakeholderrunden werden klassifiziert und mit begruendeter Entscheidung in den Dokumentstand ueberfuehrt. Der Decision Log wird als verbindliche Referenz gepflegt, inklusive Status pro Punkt (geschlossen, vertagt, abgelehnt) und nachvollziehbarer Begruendung.

Pflicht-Ergebnis von O1 ist ein inhaltlich konsistenter RFC-Stand ohne verdeckte Konflikte in Kernfragen. Was offen bleibt, wird als bewusst vertagt markiert und einem klaren Folgeprozess zugewiesen.

### Entwicklungspaket O2: Phasen- und Gate-Konsistenzpruefung
O2 stellt sicher, dass die Umsetzungslogik ueber Kapitelgrenzen hinweg widerspruchsfrei ist. Geprueft werden insbesondere die Zielbilder, Scope-Grenzen, Audit-Gates und Exit-Kriterien der Phasen A bis C sowie die Anschlussfaehigkeit zum Governance- und Migrationskapitel.

Pflicht-Ergebnis von O2 ist eine dokumentierte Konsistenzpruefung, aus der eindeutig hervorgeht, dass keine blockierenden Konflikte zwischen Public und Technical Layer bestehen und keine stillen Scope-Verschiebungen in die Umsetzung getragen werden.

### Entwicklungspaket O3: Governance-Freeze und Referenzveroeffentlichung
O3 fuehrt das formale Governance-Event fuer den Freeze durch (z. B. Proposal und Abstimmung) und ueberfuehrt den konsolidierten Stand in eine eingefrorene Referenzversion. Diese Version ist die verpflichtende Grundlage fuer die Umsetzung in Phase A bis C.

Teil von O3 ist die Festlegung einer Change-Regel nach Freeze: Welche Anpassungen als rein redaktionell gelten und welche Aenderungen nur ueber eine neue RFC-Revision und erneute Governance-Freigabe erfolgen duerfen.

### Abhaengigkeiten und Reihenfolge
Die Omega-Pakete laufen in klarer Reihenfolge. O2 setzt den inhaltlichen Abschluss aus O1 voraus. O3 setzt eine bestandene Konsistenzpruefung aus O2 voraus. Diese Reihenfolge stellt sicher, dass keine nicht-ausgeraeumten Konflikte in eine formale Freeze-Entscheidung gelangen.

### Vorlaeufige Exit-Kriterien (Omega-Go/No-Go)
- O1 ist abgeschlossen und der Decision Log ist final konsolidiert, inklusive explizit markierter Vertagungen.
- O2 bestaetigt dokumentiert die Widerspruchsfreiheit zwischen Public und Technical Layer sowie zwischen Phasen- und Gate-Logik.
- O3 ist formal beschlossen und als Freeze-Referenzversion veroeffentlicht.
- Die Change-Regel nach Freeze ist verbindlich dokumentiert.

### Out of Scope in Phase Omega
- Implementierungsarbeit in Code.
- Festlegung von Termin-, Budget- oder Ressourcencommitments.
- Vorwegnahme technischer Feindesign-Entscheidungen, die in die Umsetzungsphasen A bis C gehoeren.

## Phase A: Konsenspfad PQ-faehig
Phase A schafft die technische Entscheidungs- und Umsetzungsgrundlage fuer den PQ-faehigen Konsenspfad. Das RFC arbeitet in den Folgekapiteln bewusst mit der Praemisse, dass ein PQ-resistenter CometBFT-Fork grundsaetzlich machbar ist. Gleichzeitig ist diese Annahme in Phase A nicht gesetzt, sondern explizit pruefpflichtig. Die Machbarkeitsstudie dient als frueher Invalidierungsmechanismus: Faellt sie negativ aus, wird das RFC in der vorliegenden Form gestoppt und in eine Alternativ-Roadmap ueberfuehrt.

Die Umsetzung von Phase A erfolgt in klar abgegrenzten Entwicklungspaketen, die durch interne und/oder externe Softwareentwicklungsdienstleister geleistet werden koennen. Der Fokus liegt auf der sicheren Umstellung des Konsenspfads. Wallet-/Tx-Rollout fuer Endnutzer und der breite Ausbau des User-Client-Oekosystems sind nicht Bestandteil dieser Phase.

### Entwicklungspaket A1: Machbarkeitsstudie CometBFT-Fork (Gate 0)
A1 prueft belastbar, ob ein PQ-resistenter Konsenspfad auf Basis eines CometBFT-Forks fuer Terra Classic technisch, sicherheitlich und operativ tragfaehig ist. Die Studie umfasst nicht nur den Fork-Kern, sondern die gesamte Abhaengigkeitskette des Konsens-Stacks: primaer CometBFT, zusaetzliche in das Cosmos SDK hineinreichende Schnittstellen sowie potenzielle Auswirkungen auf den Smart-Contract-Layer inklusive wasmd-naher Integrationspunkte.

Pflicht-Ergebnis von A1 ist eine dokumentierte Go/No-Go-Bewertung mit klaren No-Go-Kriterien, Risikoklassen und Invalidierungssignalen. Ebenfalls Teil von A1 ist die strukturierte Zerlegung der Folgearbeiten in konsistente Arbeitspakete fuer Prototyp, Audit und Migration.

### Entwicklungspaket A2: PQ-Prototyp auf unabhaengigem Testnet
A2 liefert einen lauffaehigen Prototyp eines PQ-resistenten CometBFT-Forks unter Verwendung von ML-DSA. Der Prototyp soll den technischen Integrationsnachweis im realistischen Stack erbringen und deshalb die fuer den Betrieb noetigen in Cosmos SDK und wasmd hineinreichenden Abhaengigkeiten einbeziehen.

Der Prototyp wird auf einer von `rebel-2` unabhaengigen Genesis-Testnet-Chain betrieben. Die Migration einer bestehenden Live-Chain steht in diesem Paket nicht im Vordergrund. Im Zentrum stehen Funktionsfaehigkeit, Integrationskonsistenz, reproduzierbarer Betrieb und belastbare Messdaten fuer die naechsten Entscheidungen.

### Entwicklungspaket A3: Audit-Gate fuer den Prototyp (Gate 1)
A3 ist das formale Sicherheits- und Qualitaetsgate zwischen Prototyp und Migrationsentwicklung. Der Prototyp wird unabhaengig geprueft, bevor Migrationskomponenten fuer den produktiven Pfad gebaut werden.

Audit-Schwerpunkte sind Safety/Liveness-Auswirkungen, kryptografische Korrektheit der ML-DSA-Integration, deterministisches Verhalten in kritischen Konsenspfaden sowie Failure-Modes unter Last und bei fehlerhaften Eingaben. Das Ergebnis ist eine dokumentierte Go/No-Go-Entscheidung fuer den Uebergang in A4.

### Entwicklungspaket A4: Migrationskomponenten fuer den Konsenspfad
A4 entwickelt saemtliche technischen Komponenten und Betriebsstrategien, die fuer die geordnete Umstellung auf das Konsens-Zielbild erforderlich sind. Dazu gehoeren insbesondere Upgrade- und Migrations-Handler, Mechanismen zur Registrierung von PQ-Signaturen beziehungsweise PQ-Keys fuer bestehende Validatoren sowie das formale Regelwerk fuer Cutover, Aktivierung und sichere Betriebsaufnahme.

Dieses Paket stellt sicher, dass die Umstellung nicht nur technisch moeglich, sondern auch in einer produktionsnahen Governance- und Betriebslogik umsetzbar ist. Der Schwerpunkt liegt auf deterministischem Verhalten im Upgrade-Fenster und eindeutigen Regeln fuer gueltige Signaturen vor, waehrend und nach dem Cutover.

### Entwicklungspaket A5: Audit-Gate fuer Migrationskomponenten (Gate 2)
A5 ist das abschliessende Audit-Gate vor einer produktiven Umstellung des Konsenspfads. Geprueft werden Korrektheit und Manipulationsresistenz der Registrierungs- und Migrationsmechanik, Konsistenz der Cutover-Regeln sowie die sichere und deterministische Betriebsaufnahme nach Umschaltung.

Das Ergebnis ist eine belastbare Go/No-Go-Empfehlung fuer den produktiven Migrationspfad. Kritische Findings blockieren den Uebergang in nachgelagerte Rollout- und Governance-Schritte bis zur Behebung und erfolgreichem Re-Audit.

### Abhaengigkeiten und Reihenfolge
Die Pakete sind bewusst sequentiell aufgebaut. A2 setzt ein positives Ergebnis aus A1 voraus. A4 setzt ein bestandenes Audit-Gate aus A3 voraus. A5 ist verpflichtend vor jeder produktiven Migrationsentscheidung. Diese Reihenfolge verhindert, dass ungesicherte Annahmen aus fruehen Entwicklungsphasen in den Live-Migrationspfad durchrutschen.

### Vorlaeufige Exit-Kriterien (Phase-A-Go/No-Go)
- A1 ist abgeschlossen und enthaelt eine explizite Fortfuehrungsentscheidung auf Basis nachvollziehbarer No-Go-Kriterien.
- A2 liefert einen reproduzierbar lauffaehigen Prototyp auf unabhaengigem Genesis-Testnet.
- A3 ist ohne kritische Blocker fuer den Einstieg in Migrationsentwicklung bestanden.
- A4 hat die Migrationskomponenten und Cutover-Logik vollstaendig spezifiziert und implementiert.
- A5 ist ohne kritische Blocker fuer den produktiven Migrationspfad bestanden.

### Out of Scope in Phase A
- Wallet-/Tx-Rollout fuer Endnutzer.
- Aufbau des breiten PQ-nativen User-Client-Oekosystems.
- Delivery-Details wie Owner, Termine oder Budget.

## Phase B: Wallet-/Tx-Pfad PQ-faehig
Phase B schafft die technische Umsetzungsgrundlage fuer den hybriden Signaturpfad im Wallet-/Tx-Bereich. Ziel ist nicht ein frueher harter PQ-only-Cutover, sondern eine geordnete Parallelfaehigkeit von klassischer Signaturvalidierung und ML-DSA im produktionsrelevanten Transaktionspfad.

Die Ausgangslage ist gegenueber Phase A anders: Das Cosmos SDK bietet durch abstrahierte Account- und Key-Typen, flexible Datenstrukturen, anpassbare Tx-Typen und Tx-Extensions sowie einen programmierbaren Ante-Handler bereits einen grundsaetzlich geeigneten Rahmen fuer alternative Signatur- und Verifikationsmechanismen. Obwohl in der Praxis heute ueberwiegend secp256k1 fuer Tx-Signaturen genutzt wird, ist fuer Phase B daher keine vorgelagerte Machbarkeitsstudie als eigenes Invalidierungs-Gate erforderlich. Stattdessen startet die Phase mit einer koordinierten Requirements- und Architekturplanung.

### Entwicklungspaket B1: Koordinierte Planungsphase (Requirements)
B1 konsolidiert die fachlichen und technischen Anforderungen, die zur Erreichung des hybriden Signaturzielbilds notwendig sind. Im Mittelpunkt stehen die Zielarchitektur fuer Ante-Handler und SigVerification sowie die noetigen Erweiterungen bei Key-/Account-Typen, Tx-Extensions und angrenzenden Schnittstellen.

Pflicht-Ergebnis von B1 ist ein abgestimmtes Arbeitspaket-Backlog fuer Implementierung und Migration. Dazu gehoeren klar definierte Kompatibilitaetsgrenzen, Fehler- und Fallback-Verhalten sowie ein belastbarer Testansatz fuer den spaeteren Prototypbetrieb.

### Entwicklungspaket B2: Entwicklung, Umstellung und Prototyp-Testlauf
B2 implementiert den hybriden Validierungspfad mit ML-DSA-Unterstuetzung in allen betroffenen Wallet-/Tx-Komponenten entlang des realen Transaktionsflusses. Ziel ist nicht nur die Einzelintegration, sondern die konsistente Umstellung der abhaengigen Komponenten im Zusammenspiel.

Als technischer Nachweis wird ein End-to-End-Prototyp in einem gesonderten Testnet betrieben. Dieses Paket gilt als erfolgreich, wenn hybride Signaturvalidierung stabil, reproduzierbar und mit nachvollziehbarem Betriebsverhalten laeuft.

### Entwicklungspaket B3: Audit-Gate fuer den hybriden Wallet-/Tx-Pfad
B3 ist das verpflichtende Sicherheits- und Qualitaetsgate nach Implementierung und Prototypbetrieb. Geprueft werden insbesondere Korrektheit der Signaturvalidierung, Sicherheitseigenschaften im Ante- und Verifikationspfad sowie Robustheit bei Last- und Fehlerszenarien.

Das Ergebnis ist eine dokumentierte Go/No-Go-Entscheidung fuer den Uebergang in die Migrationsausarbeitung. Kritische Findings blockieren den Folgeschritt bis zur Behebung und erneuten Pruefung.

### Entwicklungspaket B4: Migrationskomponenten fuer die Einfuehrung von ML-DSA
B4 spezifiziert und implementiert die Migrationskomponenten fuer die geordnete Einfuehrung von ML-DSA im Wallet-/Tx-Pfad. Dazu gehoeren Einfuehrungsstrategie, Aktivierungslogik, Kompatibilitaetsfenster und Betriebsuebergang fuer den hybriden Betrieb.

Fuer B4 ist bewusst kein zusaetzliches verpflichtendes Audit-Gate als separater Meilenstein gesetzt. Je nach sichtbar werdender Komplexitaet koennen B3 und B4 zusammengelegt oder eng verzahnt werden, sofern die gemeinsame Go/No-Go-Logik dokumentiert bleibt.

### Abhaengigkeiten und Reihenfolge
Die Pakete sind sequentiell aufgebaut. B2 setzt ein abgeschlossenes B1-Requirementspaket voraus. B4 setzt ein bestandenes Audit-Gate aus B3 voraus, sofern B3 und B4 nicht bewusst als gemeinsamer Block gefuehrt werden. Diese Reihenfolge stellt sicher, dass die Migrationsausarbeitung auf geprueften und nicht nur implementierten Mechanismen basiert.

### Vorlaeufige Exit-Kriterien (Phase-B-Go/No-Go)
- B1 ist abgeschlossen und liefert abgestimmte Requirements inklusive Architektur- und Testgrundlage.
- B2 liefert einen reproduzierbaren End-to-End-Prototyp-Nachweis im gesonderten Testnet.
- B3 ist ohne kritische Blocker bestanden und erlaubt den Uebergang in die Migrationsausarbeitung.
- B4 ist umsetzungsreif ausgearbeitet; bei zusammengelegtem B3/B4 gilt eine gemeinsame dokumentierte Go/No-Go-Entscheidung.

### Out of Scope in Phase B
- Vollstaendiger Rollout eines Terra-Classic-eigenen User-Client-Oekosystems.
- Delivery-Details wie Owner, Termine oder Budget.

## Phase C: PQ-native Clients
Phase C ueberfuehrt die technische PQ-Faehigkeit aus dem Wallet-/Tx-Pfad in ein dauerhaft tragfaehiges Terra-Classic-eigenes User-Oekosystem. Im Gegensatz zu Phase A und B ist die Struktur hier bewusst nicht streng sequentiell. Die zentralen Arbeitspakete koennen parallel gestartet und iterativ aufeinander abgestimmt werden.

Kernziel ist, dass Terra Classic nicht von Drittparteien oder externen Releasezyklen im breiteren Oekosystem abhaengig bleibt, wenn es um PQ-faehige Wallet-, Explorer- und User-Facing-Infrastruktur geht.

### Entwicklungspaket C1: Terra-Classic-eigener Wallet-/Explorer-Stack
C1 umfasst Requirementserstellung, Ausschreibung und Beauftragung eines eigenen Wallet-/Explorer-Stacks, der den hybriden und spaeter weiterentwickelten PQ-faehigen Wallet-/Tx-Pfad vollstaendig abbildet.

Die Entwicklung kann durch externe Dienstleister erfolgen, wird jedoch als Terra-Classic-eigene Infrastruktur unter oeffentlicher Domain-Lizenzierung gefuehrt. Ziel ist ein unabhaengig betreibbarer, offen lizenzierter und nachvollziehbar wartbarer Stack fuer Retail- und Individualnutzer.

### Entwicklungspaket C2: Oeffentlich-rechtliche Traegerinstanz und Betriebsverantwortung
C2 etabliert eine oeffentlich-rechtliche Instanz, die den Stack institutionell traegt. Diese Instanz legt Anforderungen fest, beauftragt Umsetzung, verwaltet Eigentuemerschaft und verantwortet Monitoring sowie Hosting.

Der Scope umfasst nicht nur Webinstanzen und Backend-Nodes, sondern auch die Haltung und Eigentuemerschaft kritischer digitaler Assets wie DNS-Namen, Entwickleraccounts und Publikationskonten (z. B. App Stores, Paketregister, Repositories). Bestandteil von C2 ist zudem die Ausformulierung einer grundsaetzlichen verfassungsartigen Beziehung zwischen dieser Instanz und dem Terra-Classic-Governance-Modul.

### Flankierendes Entwicklungspaket C3: Externes PQ-Standards-Monitoring und Alignment
C3 laeuft als dauerhafter Parallelstrang zu C1 und C2. Ziel ist die aktive Kontakt- und Beziehungsarbeit mit relevanten Akteuren im Cosmos-, Ethereum- und breiteren Blockchain-Oekosystem, um PQ-Standardisierung frueh zu beobachten, divergierende Entwicklungen rechtzeitig zu erkennen und Terra-Classic-Entscheidungen proaktiv zu alignen.

Dieser Strang ist sicherheitsstrategisch notwendig: ML-DSA wird als aktueller Zielalgorithmus genutzt, jedoch ohne Annahme mathematischer Endgueltigkeit gegenueber kuenftigen kryptanalytischen Durchbruechen (quantum oder klassisch). Entscheidungen in Terra Classic muessen daher revisionsfaehig bleiben und auf externes Erkenntniswachstum reagieren koennen.

### Abhaengigkeiten und Parallelisierung
C1 und C2 sind grundsaetzlich parallelisierbar und muessen nicht als starre Sequenz laufen. C3 begleitet beide kontinuierlich. Wo operative Kopplungen entstehen (z. B. Release-Ownership, Betriebspflichten, Compliance-Vorgaben), werden Synchronisationspunkte als Governance-Entscheidungen dokumentiert statt als implizite Reihenfolge erzwungen.

### Vorlaeufige Exit-Kriterien (Phase-C-Go/No-Go)
- C1: Anforderungen sind verabschiedet, Beauftragung ist erfolgt, und ein lauffaehiger oeffentlich lizenzierter Wallet-/Explorer-Stack ist nachweisbar.
- C2: Die Traegerinstanz ist formal eingerichtet und uebernimmt nachweisbar Eigentuemerschaft, Betriebsverantwortung und Asset-Kontrolle (DNS, Entwickleraccounts, Distributionskanaele).
- C2: Die grundsaetzliche Beziehung der Traegerinstanz zum Terra-Classic-Governance-Modul ist verbindlich ausformuliert und veroeffentlicht.
- C3: Ein laufender externer Monitoring- und Alignment-Prozess ist etabliert, mit dokumentierten Kontakten, Bewertungszyklen und Rueckkopplung in technische Entscheidungen.

### Out of Scope in Phase C
- Detailplanung einzelner Beschaffungs- und Vergabeverfahren auf operativer Ebene.
- Vollstaendige Vorwegnahme aller kuenftigen PQ-Algorithmuswechsel; Phase C schafft die institutionelle und technische Anpassungsfaehigkeit dafuer.
