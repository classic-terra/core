# 7) Migrationspfade fuer Live-Chain

## Einordnung dieses Kapitels
Dieses Kapitel ist bewusst informativ. Es beschreibt den Cutover-Pfad fuer den Konsensbereich genauer und legt dar, wie der Umgang mit Notfallszenarien bewertet wird, falls der primaere Migrationspfad nicht sicher oder nicht rechtzeitig durchfuehrbar ist.

Es handelt sich nicht um einen Delivery-Plan, sondern um eine technische Orientierung fuer den Uebergang von bestehendem Konsensbetrieb zu PQ-basiertem Konsensbetrieb.

## Primaerer Pfad: In-Place Cutover
Der primaere Migrationspfad ist ein koordiniertes In-Place-Upgrade mit geplantem Chain-Halt und anschliessendem Restart auf dem geforkten, PQ-faehigen Konsens-Stack. Dieser Pfad priorisiert Kontinuitaet der Live-Chain und minimiert oekosystemseitige Brueche.

Voraussetzung ist, dass die vorbereitenden Sicherheitsbedingungen aus Phase A erfuellt sind, insbesondere ein gueltiges und belastbar geprueftes Binding der bestehenden Validator-Identitaeten auf neue PQ-Consensus-Keys.

## Kernelement: Binding bestehender Validatoren auf neue Consensus-Keys
Der sicherheitskritische Kern des Cutovers ist die eindeutige Zuordnung "bestehender Validator -> neuer PQ-Consensus-Key". Ohne diese Zuordnung kann nach Umschaltung nicht verifiziert werden, welcher neue Key welchem bisherigen Validator entspricht.

Das Binding wird als vorbereiteter, nachvollziehbarer Registrierungsprozess gefuehrt:
- Jeder bestehende Validator hinterlegt seinen neuen PQ-Consensus-Key ueber den vorgesehenen Registrierungsmechanismus.
- Der Prozess erzwingt Eindeutigkeit der Zuordnung und verhindert widerspruechliche Mehrfachbindungen.
- Vor dem Cutover wird ein finaler, deterministischer Snapshot der gueltigen Bindings erstellt.
- Die Cutover-Freigabe setzt eine ausreichende registrierte Voting-Power voraus (gem. Phase-A-Logik).

## Cutover-Ablauf im Konsenspfad (informativ)
1. Aktivierung des Binding-Fensters und laufende Registrierung der PQ-Consensus-Keys.
2. Pruefung der Binding-Qualitaet und Erreichen der erforderlichen Voting-Power-Schwelle.
3. Erstellung und Verifikation des finalen Binding-Snapshots.
4. Koordinierter Chain-Halt am definierten Umschaltpunkt.
5. Deployment/Start des PQ-faehigen Konsens-Stacks mit dem freigegebenen Snapshot.
6. Wiederaufnahme der Blockproduktion mit PQ-Consensus-Keys.

## Warum kein direkter Sofort-Chain-Halt ohne Binding
Ein Sofort-Halt mit direkter Umschaltung ohne vorgelagertes Binding ist kein belastbarer Pfad, weil dann die sichere Identitaetsfortschreibung der Validatoren fehlt. Das erhoeht das Risiko fuer Fehlzuordnungen, Betriebsabbrueche und Streitfaelle bei der Validator-Aktivierung nach Restart.

## Re-Genesis-Bewertung: aktuell keine belastbare Fallback-Option
Re-Genesis wird fuer Terra Classic derzeit nicht als praktikabler Fallback bewertet, sondern als operative Sackgasse. Fuer den bestehenden Chain-State im Umfang von hunderten Gigabyte ist bislang nicht belastbar nachgewiesen, dass ein sicherer und reproduzierbarer Export nach Genesis sowie ein anschliessender stabiler Import in vertretbarer Zeit moeglich ist.

Vorliegende Erfahrungsberichte deuten darauf hin, dass selbst unter extremen Anforderungen an RAM und Disk-I/O ein Genesis-Import nach langem Lauf (z. B. nach rund 24 Stunden) abgebrochen wurde. Damit fehlt derzeit der Nachweis, dass Re-Genesis unter realistischen Betriebsbedingungen als sichere Notfallstrategie taugt.

Konsequenz fuer dieses RFC: Der Migrationspfad darf nicht auf Re-Genesis als verlassbaren Rettungspfad bauen. Die Risikoreduktion muss primaer im In-Place-Cutover selbst erreicht werden (Binding-Qualitaet, Gate-Disziplin, deterministischer Snapshot, koordinierter Betriebsuebergang).

## Entscheidungskriterien und Notfallprinzipien
Die Entscheidung ueber den In-Place-Cutover folgt einem strikten Go/No-Go-Prinzip:
- Sicherheit vor Geschwindigkeit.
- Determinismus vor Ad-hoc-Workarounds.
- Dokumentierte Governance-Entscheidung vor operativem Schnellschuss.

Wenn ein Go nicht belastbar begruendet werden kann, wird kein unsicherer Cutover erzwungen. Statt auf einen aktuell nicht validierten Re-Genesis-Pfad auszuweichen, sind Nachhaertung, Re-Audit und erneute Entscheidungsrunde erforderlich.
