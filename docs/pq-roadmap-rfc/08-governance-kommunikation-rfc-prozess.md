# 8) Governance, Kommunikation und RFC-Prozess

## Governance-Entscheidungen
Dieses Kapitel definiert, wie Entscheidungen im RFC-Lebenszyklus vorbereitet, getroffen, dokumentiert und kommuniziert werden. Ziel ist ein nachvollziehbarer Prozess, der technische Qualitaet, Stakeholder-Beteiligung und formale Entscheidbarkeit verbindet.

Governance in diesem RFC bedeutet nicht operative Delivery-Steuerung. Es geht um inhaltliche Leitentscheidungen, Freigabelogik und transparente Risikoakzeptanz entlang der Phasen Omega bis C.

Verbindlich gilt, dass Nachvollziehbarkeit vor Geschwindigkeit steht, dass Risikoakzeptanz immer explizit dokumentiert wird, dass es keine phasenkritische Fortsetzung ohne erfuellte Gate-Logik gibt und dass technische Abweichungen nach dem Freeze nur ueber formale Aenderungsregeln zugelassen werden.

## Stakeholdergruppen
Fuer Rueckmeldung, Auswirkungen und Umsetzbarkeit sind folgende Gruppen zentral:

- `Nutzer`: Endanwender, deren Vermoegenssicherung, Bedienbarkeit und Migrationsfaehigkeit im Mittelpunkt der Wallet-/Client-Strategie stehen.
- `Validatoren`: Betreiber der Konsensinfrastruktur, die fuer Key-Binding, Cutover-Stabilitaet und sichere Betriebsaufnahme im Konsenspfad kritisch sind.
- `Wallet-Anbieter`: Teams und Produkte, die Signaturerzeugung, Key-Management und Recovery-Flows fuer den hybriden und spaeter PQ-nativen Betrieb umsetzen.
- `Exchanges`: zentrale Handels- und Abwicklungsakteure mit hohen Sicherheitsanforderungen und frueher Integrationsrelevanz fuer PQ-faehige Signaturpfade.
- `Custody-Anbieter`: professionelle Verwahrer mit erhoehter Verantwortung fuer Schluesselkontrolle, operative Sicherheit und revisionsfaehige Prozesse.
- `Integratoren`: Infrastruktur- und Produktteams, die Nodes, APIs, SDK-nahe Dienste und Betriebsprozesse mit den neuen Pfaden kompatibel halten muessen.
- `Externe Infrastruktur- und Standardisierungsakteure`: Referenzquellen aus dem breiteren Oekosystem (z. B. Cosmos-/Ethereum-nahe Projekte, Security-Forschung, Tooling-Anbieter), die fuer Phase C und das fortlaufende PQ-Alignment relevant sind.

## Rollen im RFC-Prozess
Der Prozess unterscheidet funktionale Rollen: RFC-Editoren konsolidieren den Text und pflegen Versionen sowie Change Log, technische Maintainer bewerten Koharenz, Risiken und Gate-Anschlussfaehigkeit, die Governance-Instanz trifft formale Freigabe- und Freeze-Entscheidungen, und Auditoren liefern interne oder externe Gate-Bewertungen gemaess Kapitel 06. Rollen koennen personell ueberschneiden; die Protokollierung muss dennoch eindeutig trennen, wer vorgeschlagen, geprueft und final entschieden hat.

## RFC-Feedbackzyklus
Der Feedbackprozess laeuft iterativ bis zur Freeze-Reife und folgt einem stabilen Vier-Schritt: erstens wird eine RFC-Version veroeffentlicht und ein Feedbackfenster geoeffnet, zweitens werden Kommentare klassifiziert, drittens wird eine Folgerevision mit Change Log publiziert, viertens folgt eine kurze Reviewrunde fuer die geaenderten Abschnitte. Jede Runde endet mit einem transparenten Delta, das geaenderte Punkte, bewusst offene Punkte und die Uebernahme in den Decision Log klar ausweist.

## Freeze
Der RFC-Freeze ist als vorgelagerte `Phase Omega` definiert und muss abgeschlossen sein, bevor die Umsetzungsphasen A-C beginnen.

Ausgeloest wird der Freeze durch ein formales Governance-Proposal zur Freeze-Entscheidung. Freeze-reif ist der RFC erst dann, wenn kritische offene Entscheidungen geklaert oder begruendet vertagt sind, keine blockierenden Widersprueche zwischen Public und Technical Layer bestehen, Priorisierung und Phasenlogik stabil sind, die Audit-Gate-Logik je Phase vollstaendig dokumentiert ist und Stakeholder-Feedback inklusive Scope- und Decision-Log-Pflege sauber eingearbeitet wurde. Das Freeze-Ergebnis ist eine als "frozen" markierte Referenzversion; Aenderungen an eingefrorenen Kernentscheidungen sind nur ueber eine neue RFC-Revision zulaessig.

## Change-Control nach Freeze
Nach dem Freeze gilt ein zweistufiges Aenderungsmodell. Redaktionelle Aenderungen ohne Wirkung auf Zielbild, Gate-Logik oder Scope koennen direkt eingepflegt werden. Inhaltliche Aenderungen, die Zielbilder, Phasenlogik, Gate-Kriterien, Risikoannahmen oder Migrationsprinzipien beruehren, erfordern eine neue RFC-Revision mit formaler Governance-Freigabe. Jede inhaltliche Aenderung muss die betroffenen Kapitel explizit referenzieren, mindestens ueber Zielbild, Roadmap-Phase, Audit-Gate und Decision-Log-Eintrag.

## Entscheidungs- und Eskalationslogik waehrend der Umsetzung
Auch nach dem Freeze bleiben Governance-Entscheidungen notwendig, insbesondere bei kritischen Audit-Findings mit Risikoakzeptanzbedarf, Gate-Blockern mit Re-Planungsbedarf, externen PQ-Entwicklungen mit Wirkung auf bestehende Annahmen und Abweichungen von definierten Cutover- oder Sicherheitsprinzipien. Die Eskalationslogik ist strikt: zuerst technische Klaerung und Dokumentation, danach explizite Governance-Entscheidung mit Go/No-Go oder Vertagung. Eine stille Fortsetzung mit ungeklaerten kritischen Abweichungen ist ausgeschlossen.

## Kommunikationsrahmen
Die Kommunikation folgt dem Prinzip "ein Prozess, zwei Ebenen": Im Public Layer stehen Zweck, Risiko und Auswirkungen verstaendlich im Vordergrund; im Technical Layer stehen Entscheidungsgrundlagen, Gate-Status und Rest-Risiken praezise im Vordergrund. Bei jeder phasenrelevanten Entscheidung sind mindestens Entscheidung und Geltungsbereich, technische Begruendung, betroffene Phasen beziehungsweise Gates sowie offene Rest-Risiken mit naechstem Pruefpunkt zu kommunizieren.

## Verbindung zum Decision Log
Dieses Kapitel definiert den Prozess, Kapitel 09 ist das operative Register dazu. Jeder relevante Governance-Schritt muss im Decision Log einen nachvollziehbaren Statuswechsel erzeugen (`offen`, `geschlossen`, `vertagt`) inklusive kurzer Begruendung.
