# 4) Zielbilder und Optionen (Technical Layer)

## Zielbild Konsens
Fuer den Konsenspfad ist das Zielbild ein klarer PQ-only-Betrieb mit ML-DSA ab einem definierten Umschalttag (Tag X). Diese Festlegung bedeutet, dass der Konsenspfad nach dem Cutover nicht dauerhaft mit zwei unterschiedlichen Signaturregimen parallel betrieben werden soll, sondern einen eindeutigen Zielzustand besitzt.

Um das Zielbild nachvollziehbar zu machen, betrachtet das RFC zwei Optionen. Option A ist der direkte Cutover in einen PQ-only-Konsensmodus ab Tag X. Option B ist ein Hybridmodell als Uebergangsmodus, in dem klassische und neue Signaturverfahren im Konsenspfad zeitlich begrenzt parallel akzeptiert werden.

Das Hybridmodell wird im Konsens nicht als Zielbild uebernommen, weil es die Regelbasis im sicherheitskritischsten Pfad erweitert und damit Komplexitaet, Angriffsoberflaeche und Koordinationsrisiko erhoeht. Zusaetzlich entsteht im Hybridbetrieb ein erweiterter Test-, Audit- und Betriebsaufwand, da mehrere gueltige Signaturmodi gleichzeitig korrekt und deterministisch verarbeitet werden muessen. Das RFC priorisiert daher einen klaren Endzustand mit eindeutigem Verifikationsverhalten.

## Zielbild Wallet-/Tx-Pfad
Im Wallet-/Tx-Pfad ist das Zielbild ein hybrider Uebergangspfad statt eines fruehen harten PQ-only-Cutovers. Klassische und PQ-resistente Signaturen werden in einer definierten Uebergangsphase parallel unterstuetzt, damit die Adoption an Sicherheitsbedarf, technische Reife und operative Faehigkeit der jeweiligen Teilnehmer angepasst werden kann.

Der Schwerpunkt liegt deshalb auf geordneter Einfuehrung, klaren Kompatibilitaetsgrenzen und einer nachvollziehbaren Umstellungslogik fuer Nutzer und Integratoren. In der fruehen Phase kann die Unterstuetzung PQ-resistenter Verfahren in Client- und nutzernahen Umgebungen noch lueckenhaft sein, waehrend UX, Tooling und Recovery-Prozesse erst in spaeteren RFC-Phasen ausreifen.

Das Zielbild priorisiert daher eine gestufte Adoption: sicherheitskritische und technisch versierte Akteure wie CEXs, Custody-Anbieter und Infrastrukturprojekte koennen PQ-Signaturen frueh produktiv einsetzen und ihre Systeme fruehzeitig absichern. Weniger technisch versierte Retail-Teilnehmer koennen schrittweise folgen, sobald Wallet-Oekosystem, Standards und Bedienbarkeit ausreichend gereift sind.

## Zielbild PQ-native Clients
Das Zielbild fuer PQ-native Clients ist der langfristige Aufbau eines Terra-Classic-eigenen Wallet- und User-Facing-Client-Oekosystems mit nativer PQ-Kompatibilitaet. Signaturerzeugung, Signaturpruefung und Schluesselverwaltung sollen fuer Endnutzer frueh verfuegbar und praktikabel werden, statt auf externe Reifegrade im breiteren Cosmos- oder Blockchain-Umfeld warten zu muessen.

Der Aufbau erfolgt unter aktiver Einbindung von Infrastrukturakteuren, die im Terra-Classic-Oekosystem bereits produktiv aktiv sind und operative Reife bewiesen haben. Damit wird die Einfuehrung PQ-kompatibler Wallet-Systeme nicht von Drittparteien, grossen Oekosystemakteuren oder extern gesetzten Standards im noch dynamischen PQ-Umfeld abhaengig gemacht.

Das Zielbild ist, dass Individualakteure und Retail-Teilnehmer innerhalb von Terra Classic so frueh wie moeglich an PQ-freundlichen Signaturverfahren teilnehmen koennen, um ihre Vermoegenswerte abzusichern, ohne von der möglicherweise stockenden PQ-Adoption ausserhalb von Terra Classic blockiert zu sein.

## Bewusst offene Punkte
Trotz der Festlegung zentraler Leitentscheidungen bleiben in diesem Kapitel bewusst Punkte offen, die erst mit fortschreitender technischer Ausarbeitung und Stakeholder-Abstimmung final entschieden werden sollten. Dazu gehoeren insbesondere konkrete Integrationsdetails je Oekosystemkomponente, exakte Uebergangsbedingungen einzelner Teilpfade und die Feingranularitaet der Rollout- und Kommunikationslogik.

Alle offenen Punkte werden nicht implizit, sondern explizit im Decision Log gefuehrt und dort als offen, geschlossen oder vertagt markiert. Damit bleibt transparent, welche Zielbilder bereits verbindlich sind und an welchen Stellen noch Entscheidungsbedarf besteht.
