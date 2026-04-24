# 1) Executive Summary

## Kontext und Zweck
Terra Classic nutzt heute kryptografische Signaturen in mehreren sicherheitskritischen Bereichen. Diese Signaturen stellen sicher, dass nur berechtigte Akteure gueltige Aktionen ausfuehren koennen und dass die Chain einen gemeinsamen, verifizierbaren Zustand behaelt. Post-Quantum-Kryptografie (kurz: PQ, also kryptografische Verfahren mit Blick auf kuenftige Quantencomputer-Risiken) wird relevant, weil langfristig leistungsfaehige Quantencomputer klassische Signaturalgorithmen angreifen und brechen koennten, die heute breit eingesetzt werden.

Dieses RFC schafft dafuer einen strukturierten Orientierungsrahmen. Es definiert, wie Migrationspfade identifiziert, bewertet und priorisiert werden, und beschreibt den Rahmen fuer Audit-Gates, Governance und Feedback bis zum RFC-Freeze. Gleichzeitig ist diese Fassung bewusst keine technische Feinspezifikation und auch kein Release-, Ressourcen- oder Termincommitment.

## Aufbau des Dokuments
Im Terra-Classic-Stack werden Signaturen in mehreren Bereichen verwendet. Diese Bereiche werden in diesem Dokument als Pfade bezeichnet. Das RFC ordnet diese Pfade zunaechst auf hoher Ebene ein und arbeitet die genaue Abgrenzung sowie ihre jeweilige Kritikalitaet in den Folgekapiteln systematisch aus. Auf dieser Basis wird die Priorisierung transparent hergeleitet und begruendet.

Nach der Executive Summary folgt die Ausgangslage mit Begriffsfundament. Darauf bauen Zielbilder und Optionen auf, bevor die Roadmap in Phasen strukturiert wird. Anschliessend werden Audit-Gates, Migrationspfade fuer die Live-Chain sowie Governance- und Feedbackprozess beschrieben. Den Abschluss bildet ein Decision Log, in dem geschlossene und offene Punkte nachvollziehbar gefuehrt werden.

## Leselogik und Abgrenzung
Das RFC ist in Public Layer und Technical Layer gegliedert. Der Public Layer erklaert Motivation, Struktur und Entscheidungslogik in gut nachvollziehbarer Sprache. Der Technical Layer vertieft Pfade, Risiken, Pruefpfade und Entscheidungsoptionen. Beide Ebenen sind aufeinander abgestimmt, damit Leser je nach Detailtiefe einsteigen koennen, ohne den roten Faden zu verlieren.

Diese Executive Summary nimmt keine technischen Detailentscheidungen vorweg und legt keine Implementierungsabfolge im Feinschnitt fest. Sie dient als Lesefuehrung. Die inhaltlichen Ausarbeitungen, Begruendungen und Entscheidungsvorlagen folgen in den nachgelagerten Kapiteln.
