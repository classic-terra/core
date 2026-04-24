# 3) Ausgangslage und Begriffe

## Welche Signaturpfade im RFC betrachtet werden
Bevor der Status quo beschrieben wird, grenzt dieses Kapitel die Signaturpfade ab, die im weiteren Verlauf des RFC betrachtet werden. Der Begriff Pfad meint dabei keinen einzelnen Code-Abschnitt, sondern einen zusammenhaengenden technischen und betrieblichen Wirkbereich, in dem Signaturen erstellt, uebertragen, geprueft und als Entscheidungsgrundlage verwendet werden.

Fuer Terra Classic sind vor allem drei Perspektiven relevant. Erstens der Konsenspfad, in dem Validator-Signaturen direkt die gemeinsame Chain-Wahrheit absichern. Zweitens der Wallet-/Tx-Pfad, in dem Nutzertransaktionen signiert und im Backend validiert werden. Drittens der user-facing Clientpfad, in dem Signaturablaeufe fuer Endnutzer sichtbar und bedienbar werden. Im weiteren Verlauf dieses Kapitels liegt der Status-quo-Fokus auf den beiden technisch-kritischen Kernpfaden Konsens sowie Wallet/Tx; der Clientpfad wird spaeter in den Zielbildern und Roadmap-Phasen vertieft.

Diese Abgrenzung ist wichtig, weil die Pfade unterschiedliche Risikoprofile und Migrationsaufwaende haben. Ein Fehler im Konsenspfad wirkt systemisch auf die Chain-Stabilitaet, waehrend Fehler im Wallet-/Tx-Pfad haeufiger als Integrations-, Kompatibilitaets- oder Bedienungsprobleme sichtbar werden. Das RFC behandelt deshalb nicht nur "Signaturen" im Allgemeinen, sondern explizit Signaturen in ihrem jeweiligen Betriebskontext.

## Status quo des Konsenspfads
Im Konsenspfad sichern Signaturen die Blockproduktion und Finalitaet zwischen Validatoren. Signaturpruefungen in Vote-, Proposal- und Commit-bezogenen Ablaeufen gehoeren damit zur sicherheitskritischen Grundfunktion des Netzwerks. Wenn in diesem Bereich die Verifikation nicht mehr konsistent oder deterministisch funktioniert, betrifft das nicht nur einzelne Teilnehmer, sondern potenziell die Stabilitaet des gesamten Systems.

Im aktuellen Terra-Classic-Kontext ist im Konsenspfad insbesondere **Ed25519** als relevantes Verfahren sichtbar. Der operative Rahmen des Konsenspfads umfasst nicht nur den reinen Verifikationscode, sondern auch die angebundenen Komponenten fuer Validator-Betrieb, Schluesselhaltung und Signaturbereitstellung. Daraus folgt fuer den RFC eine klare Ausgangslage: Konsensseitige Signaturaenderungen muessen immer unter den Gesichtspunkten Safety, Liveness, Upgrade-Determinismus und Betriebsstabilitaet bewertet werden. Aus diesem Grund wird der Konsenspfad in der weiteren Roadmap als hochkritischer Pfad behandelt.

## Status quo des Wallet-/Tx-Pfads
Im Wallet-/Tx-Pfad werden Signaturen fuer Nutzertransaktionen erstellt und geprueft. Dieser Pfad ist stark oekosystemgepraegt: Wallets, Custody-Umgebungen, Exchanges, APIs, SDKs und Integrationen muessen in der Praxis konsistent zusammenspielen. Die Signaturpruefung ist hier zwar ebenfalls sicherheitsrelevant, ihre Auswirkungen zeigen sich jedoch haeufig zuerst in Kompatibilitaet, Nutzerfluss und Supportaufwand.

Im aktuellen Terra-Classic-Kontext ist im Wallet-/Tx-Pfad insbesondere **secp256k1** als relevantes Verfahren sichtbar. Die Ausgangslage ist deshalb eine andere als im Konsenspfad. Im Wallet-/Tx-Bereich ist nicht nur die kryptografische Korrektheit entscheidend, sondern auch die Frage, wie breit ein neues Signaturverhalten in bestehende Toolchains und Oberflaechen integriert werden kann. Fuer den RFC bedeutet das: Dieser Pfad braucht neben Sicherheitsargumentation immer auch Integrations- und Betriebsargumentation.

## Warum diese Ausgangslage fuer die Roadmap entscheidend ist
Die Roadmap wird auf dieser Status-quo-Analyse aufgebaut. Erst die klare Trennung der Pfade erlaubt es, Prioritaet, Risiko und Migrationslogik nachvollziehbar zu begruenden. Ohne diese Trennung wuerden technische Entscheidungen leicht pauschalisiert und damit schwer vergleichbar.

Dieses Kapitel legt daher das gemeinsame Fundament fuer die nachfolgenden Kapitel: Zielbilder und Optionen bauen auf derselben Pfadabgrenzung auf, Audit-Gates orientieren sich an denselben Risikoprofilen, und auch Governance-Entscheidungen werden entlang dieser Struktur bewertet.

## Begriffe fuer die Folgekapitel
- **Pfad** - Ein Pfad bezeichnet im RFC einen zusammenhaengenden Wirkbereich aus Technik und Betrieb, in dem Signaturen erzeugt, uebertragen und geprueft werden.
- **Konsenspfad** - Der Konsenspfad umfasst Signatur- und Verifikationsablaeufe, die direkt die gemeinsame Chain-Wahrheit absichern. Das betrifft insbesondere die gemeinsame und koordinierte Absicherung der Blockprüfung und -produktion.
- **Wallet-/Tx-Pfad** - Der Wallet-/Tx-Pfad umfasst nutzerseitige Transaktionssignaturen und deren Pruefung in den Backend- und Protokollablaeufen. Diese Signaturen sichern ab, dass Transaktionen vom Einzelnutzer authorisiert wurden und sichern daher einzelne Nutzer vor der Manipulation ihres Kontos ab.
- **Clientpfad** - Der Clientpfad bezeichnet die nutzernahe Schicht, in der Signaturprozesse in Wallets und Oberflaechen sichtbar und bedienbar werden. Clientseitig ist weniger die Prüfung relevant als mehr die _Erzeugung_ von Signaturen mit clientseitig verfügbaren Algorithmen.
- **Kryptografische Signatur** - Eine kryptografische Signatur ist ein digitaler Nachweis, der Absenderbindung und Inhaltsbindung herstellt.
- **Signaturverifikation** - Signaturverifikation ist die Pruefung, ob Nachricht, Signatur und oeffentlicher Schluessel logisch zusammenpassen.
- **Privater Schluessel** - Der private Schluessel ist der geheime Schluessel einer Partei und wird zur Signaturerzeugung verwendet.
- **Oeffentlicher Schluessel** - Der oeffentliche Schluessel ist der verteilbare Gegenpart zum privaten Schluessel und wird fuer die Signaturpruefung genutzt.
- **Authentizitaet** - Authentizitaet bedeutet, dass eine Nachricht tatsaechlich vom behaupteten Absender stammt.
- **Integritaet** - Integritaet bedeutet, dass der Inhalt einer Nachricht seit der Signatur nicht unbemerkt veraendert wurde.
- **Validator** - Ein Validator ist ein Netzwerkteilnehmer, der im Konsensprozess Signaturen erzeugt und prueft.
- **Voting Power (VP)** - Voting Power ist das Konsensgewicht eines Validators.
- **Safety** - Safety bedeutet im Konsenskontext, dass das Netzwerk keine widerspruechlichen gueltigen Zustaende akzeptiert.
- **Liveness** - Liveness bedeutet, dass das Netzwerk unter gueltigen Betriebsannahmen weiter Bloecke produziert und nicht dauerhaft stehenbleibt.
- **Upgrade-Determinismus** - Upgrade-Determinismus bedeutet, dass ein Protokoll- oder Softwarewechsel unter denselben Bedingungen bei allen Teilnehmern zu denselben Ergebnissen fuehrt.
- **Cutover** - Ein Cutover ist ein geplanter Umschaltpunkt zwischen zwei Betriebsmodi.
- **Audit-Gate** - Ein Audit-Gate ist ein verpflichtender Pruef- und Freigabepunkt vor dem Uebergang in die naechste Phase.
- **In-Place Upgrade** - Ein In-Place Upgrade ist ein Migrationspfad, bei dem die bestehende Live-Chain kontrolliert weitergefuehrt und technisch umgestellt wird.
- **Re-Genesis** - Re-Genesis ist der Fallback-Pfad, falls ein In-Place Upgrade nicht tragfaehig ist.
- **Key-Binding** - Key-Binding bezeichnet die nachvollziehbare Zuordnung eines neuen Schluessels zu einer bestehenden Validator-Identitaet.
- **CometBFT** - CometBFT ist eine Softwarekomponente von Terra Classic, in der der zentrale Konsens- und Validator-Signaturabläufe verankert sind.
- **Ante** - Ante bezeichnet im SDK-Kontext den Vorpruefungsbereich fuer Transaktionen vor der eigentlichen Ausfuehrung.
- **RFC** - RFC steht fuer "Request for Comments" und bezeichnet in diesem Kontext ein strukturiertes Diskussions- und Entscheidungsdokument, das Richtung, Optionen, Risiken und offene Punkte transparent macht, bevor verbindliche Delivery-Entscheidungen getroffen werden.
- **Ed25519** - Ed25519 ist im aktuellen Terra-Classic-Stand das relevante Signaturverfahren im Konsenspfad.
- **secp256k1** - secp256k1 ist im aktuellen Terra-Classic-Stand das relevante Signaturverfahren im Wallet-/Tx-Pfad.
- **PQ (Post-Quantum)** - PQ steht fuer kryptografische Verfahren, die gegen Bedrohungsmodelle mit leistungsfaehigen Quantencomputern robuster sein sollen als viele heute verbreitete klassische Verfahren.
- **ML-DSA** - ML-DSA ist im vorliegenden Plan als Zielprofil fuer den Konsenspfad festgelegt.
