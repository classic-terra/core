# 2) Kryptografische Signaturen einfach erklaert (Public Bridge)

## Worum es bei einer Signatur geht
Eine kryptografische Signatur ist ein digitaler Nachweis, dass eine Nachricht wirklich von der Person oder Instanz stammt, die als Absender genannt ist, und dass der Inhalt unterwegs nicht unbemerkt veraendert wurde. Praktisch bedeutet das: Wer eine signierte Nachricht prueft, kann nachvollziehen, ob Absender und Inhalt zusammenpassen.

In einem Blockchain-System ist genau dieser Nachweis zentral. Validatoren, Wallets und Anwendungen treffen laufend Entscheidungen auf Basis signierter Daten. Wenn dieser Nachweis nicht mehr verlaesslich ist, wird aus technischer Unsicherheit sehr schnell ein Betriebs- und Vertrauensproblem.

Ohne Signaturen koennte jede Partei beliebig behaupten, eine Nachricht stamme von jemand anderem. Mit Signaturen laesst sich dagegen pruefen, ob eine Nachricht autorisiert ist. Das schuetzt nicht nur einzelne Transaktionen, sondern auch die gemeinsame Verlaesslichkeit des Netzwerks.

In der Praxis werden dafuer komplexe kryptografische Verfahren eingesetzt. Diese Verfahren sind deutlich anspruchsvoller als ein einfaches Lehrbeispiel und genau darauf ausgelegt, Faelschungen praktisch unmoeglich zu machen.

## Asymmetrische Signatur
Im Blockchain-Kontext geht es meist um Signaturen asymmetrischer kryptografischer Verfahren. Das bedeutet, dass eine Partei zwei Schluessel besitzt: einen privaten Schluessel und einen oeffentlichen Schluessel. Der private Schluessel bleibt geheim und wird nicht weitergegeben. Der oeffentliche Schluessel wird aus dem privaten Schluessel abgeleitet und kann im Netzwerk offen verteilt werden.

Entscheidend ist: Die Ableitung ist so konstruiert, dass sie praktisch nicht einfach rueckwaerts funktioniert. Aus dem oeffentlichen Schluessel soll man den privaten Schluessel nicht herleiten koennen. Gleichzeitig bleibt der Zusammenhang zwischen beiden Schluesseln stark genug, damit Signaturen geprueft werden koennen.

## Alice/Bob-Positivfall
Alice sendet Bob eine Nachricht und signiert sie mit ihrem privaten Schluessel. Die Signatur wird gemeinsam mit der Nachricht uebertragen. Fuer Bob ist dann pruefbar, ob Nachricht und Signatur zusammenpassen und ob die Nachricht zu Alices oeffentlichem Schluessel gehoert.

Anschaulich kann man sich die Signatur wie eine spezielle Pruefzahl vorstellen, die aus zwei Teilen entsteht: aus dem Inhalt der Nachricht und aus Alices privatem Schluessel. Genau dadurch werden Identitaet und Integritaet miteinander verbunden. Alices Identitaet steckt im Schluesselbezug, die Integritaet im Nachrichtenteil. Wenn sich einer der beiden Teile aendert, passt die Pruefung nicht mehr.

Bob prueft also Signatur, Nachricht und Alices oeffentlichen Schluessel gemeinsam. Wenn die Pruefung erfolgreich ist, kann Bob davon ausgehen, dass die Nachricht von Alice stammt und seit der Signatur nicht veraendert wurde. Dieser Ablauf ist der Normalfall: Signaturpruefung schafft Verbindlichkeit zwischen Absender, Inhalt und Empfaenger.

Im Negativfall versucht ein Angreifer, sich zwischen Alice und Bob zu schalten und als Alice aufzutreten. Wenn Signaturen nicht mehr sicher waeren oder erfolgreich gefaelscht werden koennten, wuerde Bob unter Umstaenden eine manipulierte Nachricht als echt akzeptieren. Die Folge waere nicht nur ein Fehler in einer einzelnen Nachricht, sondern ein grundsaetzlicher Vertrauensverlust in die Kommunikation des Netzwerks.

## Bruecke zum PQ-Thema
Der zentrale Punkt dieses RFC ist nicht, konkrete Verfahren im Detail zu erklaeren, sondern die Sicherheitsannahmen hinter realen Signaturverfahren sichtbar zu machen. Klassische Verfahren gelten heute bei ueblichen Parametern als praktisch sicher. Mit dem Aufkommen leistungsfaehiger Quantencomputer kann sich diese Annahme fuer bestimmte Verfahren jedoch aendern.

Genau daraus entsteht die Notwendigkeit, die technischen Bausteine von Terra Classic, die auf Signaturverifikation basieren, strukturiert auf neue Verfahren umzustellen: sogenannte PQ-Verfahren (Post-Quantum-Verfahren).
