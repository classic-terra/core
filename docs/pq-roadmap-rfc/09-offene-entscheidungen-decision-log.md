# 9) Offene Entscheidungen (Decision Log)

Dieses Kapitel ist das operative Entscheidungsregister des RFC. Es fuehrt geschlossene, offene und vertagte Punkte in einer Form, die fuer Governance, Audit und Phasensteuerung direkt nutzbar ist. Der Decision Log wird bei jeder relevanten RFC-Revision aktualisiert und ist die verbindliche Referenz fuer Statuswechsel.

## Geschlossene Leitentscheidungen

### G-01
**Thema:** Konsens-Crypto-Profil  
**Entscheidung:** ML-DSA als Zielprofil im Konsenspfad.  
**Status:** geschlossen  
**Referenz:** Kapitel 04, 05  
**Kurzbegruendung:** Verbindliches PQ-Zielbild fuer den sicherheitskritischsten Pfad.

### G-02
**Thema:** Konsens-Migrationsmodus  
**Entscheidung:** Kein dauerhafter Hybridbetrieb im Konsens; klarer Cutover auf PQ-only-Zustand.  
**Status:** geschlossen  
**Referenz:** Kapitel 04, 05  
**Kurzbegruendung:** Reduziert Regelkomplexitaet, Angriffsoberflaeche und Betriebsrisiko.

### G-03
**Thema:** Konsens-Cutover-Mechanik  
**Entscheidung:** Key-Binding bestehender Validatoren auf PQ-Consensus-Keys plus Voting-Power-Schwelle vor Halt.  
**Status:** geschlossen  
**Referenz:** Kapitel 05, 07  
**Kurzbegruendung:** Eindeutige Identitaetsfortschreibung als Voraussetzung fuer sicheren Restart.

### G-04
**Thema:** Wallet-/Tx-Zielbild  
**Entscheidung:** Hybrider Signaturpfad mit gestufter Adoption statt fruehem PQ-only-Cutover.  
**Status:** geschlossen  
**Referenz:** Kapitel 04, 05  
**Kurzbegruendung:** Oekosystemsensitiver Rollout mit frueher Nutzbarkeit fuer technisch starke Akteure.

### G-05
**Thema:** Client-Zielbild  
**Entscheidung:** Aufbau eines Terra-Classic-eigenen Wallet-/Explorer-Oekosystems mit nativer PQ-Kompatibilitaet.  
**Status:** geschlossen  
**Referenz:** Kapitel 04, 05  
**Kurzbegruendung:** Reduziert Abhaengigkeit von Drittparteien und externen Releasezyklen.

### G-06
**Thema:** RFC-Freeze-Logik  
**Entscheidung:** Vorgelagerte Phase Omega als formale Startfreigabe vor A-C.  
**Status:** geschlossen  
**Referenz:** Kapitel 05, 08  
**Kurzbegruendung:** Sichert inhaltliche Baseline und verhindert Scope-Drift waehrend Umsetzung.

### G-07
**Thema:** Re-Genesis-Bewertung  
**Entscheidung:** Re-Genesis aktuell nicht als belastbarer Fallback, sondern als operative Sackgasse zu behandeln.  
**Status:** geschlossen  
**Referenz:** Kapitel 07  
**Kurzbegruendung:** Fehlender Nachweis fuer sicheren, reproduzierbaren Export/Import bei grossem State.

### G-08
**Thema:** Audit-Gate-Grundsatz  
**Entscheidung:** Gate-Pflicht entlang der definierten Gate-Matrix mit Blocker-/Re-Audit-Regeln.  
**Status:** geschlossen  
**Referenz:** Kapitel 06  
**Kurzbegruendung:** Erzwingt risikoadaequate Uebergaenge zwischen den Phasen.

## Offene Architektur- und Prozessentscheidungen

### O-01
**Entscheidungsfrage:** Exakte Ausgestaltung des Wallet-/Tx-Hybridpfads (Keytypen, Kompatibilitaetsfenster, Aktivierungslogik).  
**Optionen (Arbeitsstand):** konservativer Einstieg / gestufte Aktivierung / aggressiver Rollout.  
**Bewertungskriterien:** Sicherheit, Integrationsaufwand, UX-Risiko, Betriebskomplexitaet.  
**Zielphase:** B1  
**Status:** offen

### O-02
**Entscheidungsfrage:** Formale Trigger fuer Cutover-Go/No-Go im Konsenspfad (jenseits Mindestschwellen).  
**Optionen (Arbeitsstand):** starre Kriterien / gewichtetes Kriterienmodell / mehrstufige Freigabe.  
**Bewertungskriterien:** Safety/Liveness-Risiko, Determinismus, operative Steuerbarkeit.  
**Zielphase:** A4-A5  
**Status:** offen

### O-03
**Entscheidungsfrage:** Governance-Form der Traegerbeziehung in Phase C.  
**Optionen (Arbeitsstand):** direkte Governance-Anbindung / mandatierte Unterstruktur / hybrides Modell.  
**Bewertungskriterien:** Rechenschaft, Handlungsfaehigkeit, Asset-Kontrolle, Transparenz.  
**Zielphase:** C2  
**Status:** offen

### O-04
**Entscheidungsfrage:** Regime fuer zusaetzliche Re-Audits bei B3/B4-Zusammenlegung.  
**Optionen (Arbeitsstand):** verpflichtend bei Triggern / fallweise Governance-Entscheidung.  
**Bewertungskriterien:** Sicherheitswirkung, Zeitbedarf, Pruefaufwand.  
**Zielphase:** B3-B4  
**Status:** offen

### O-05
**Entscheidungsfrage:** Externe PQ-Alignment-Mechanik (Turnus, Entscheidungsimpact).  
**Optionen (Arbeitsstand):** periodischer Zyklus / ereignisgetrieben / kombiniert.  
**Bewertungskriterien:** Reaktionsfaehigkeit, Prozesslast, Entscheidungsqualitaet.  
**Zielphase:** C3  
**Status:** offen

## Vertagte Entscheidungen

### V-01
**Thema:** Konkretisierung eines Re-Genesis-Notfallpfads.  
**Vertagungsgrund:** Aktuell kein belastbarer technischer Nachweis fuer sicheren Export/Import auf Terra-Classic-Stategroesse.  
**Re-Trigger fuer Wiederaufnahme:** Belastbare End-to-End-Nachweise unter realistischen Betriebsbedingungen.  
**Status:** vertagt

### V-02
**Thema:** Endgueltiges Langfristprofil nach ML-DSA (post-ML-DSA-Optionen).  
**Vertagungsgrund:** Externe PQ-Entwicklung und Kryptanalyse noch in Bewegung.  
**Re-Trigger fuer Wiederaufnahme:** Relevante neue Forschungsergebnisse oder Standardaenderungen mit konkretem Risikoeinfluss.  
**Status:** vertagt

## Statusschema und Statuswechselregeln
Statuswerte:
- `offen`: Frage ist entscheidungsreif beschrieben, aber noch nicht final beschlossen.
- `geschlossen`: Frage ist final entschieden und als aktuelle Referenz gueltig.
- `vertagt`: Frage ist bewusst zurueckgestellt und an einen klaren Re-Trigger gebunden.

Statuswechsel:
- `offen -> geschlossen` nur mit dokumentierter Governance-Entscheidung und Kapitelreferenz.
- `offen -> vertagt` nur mit begruendetem Vertagungsgrund und explizitem Re-Trigger.
- `vertagt -> offen` sobald der definierte Re-Trigger eingetreten ist.
- Ruecknahme einer `geschlossenen` Leitentscheidung nur ueber RFC-Revision gemaess Freeze-/Change-Control-Regeln aus Kapitel 08.
