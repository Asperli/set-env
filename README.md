# Arbeitsplatz festlegen

Kleines Fenster, mit dem ein Arbeitsplatz gewählt wird. Ein Klick schreibt den zugehörigen Wert in eine **Benutzervariable** des angemeldeten Benutzers. Dafür sind keine Administratorrechte nötig.

Der Fenstertitel kommt aus der Konfiguration. Der gewählte Arbeitsplatz ist grün, die übrigen Buttons sind grau.

Das Programm läuft unter Windows.

## Nach dem Klonen

Zum Bauen wird [Go](https://go.dev/dl/) gebraucht, Version 1.27.4 oder neuer. Ein C-Compiler ist nicht nötig. Die Datei `rsrc_windows_amd64.syso` bleibt im Ordner. Go bindet sie beim Build ein, damit das Fenster die Windows-Darstellung und die DPI-Einstellung übernimmt.

```text
git clone https://github.com/Asperli/set-env.git
cd set-env
go build -ldflags="-H windowsgui" -o Set-Env.exe .
```

`-H windowsgui` verhindert, dass zusätzlich ein Konsolenfenster aufgeht. `Set-Env.exe` entsteht lokal und liegt nicht im Repository.

`set-env.config.yaml` liegt schon neben dem Quellcode. Die exe und diese Datei müssen im selben Ordner bleiben. Danach die exe per Doppelklick starten.

Eine geänderte Konfiguration braucht keinen neuen Build. Nur das Fenster neu öffnen. Ein neuer Build ist nötig, wenn sich der Go-Code geändert hat.

## Start

`Set-Env.exe` und `set-env.config.yaml` liegen im selben Ordner. Die exe per Doppelklick starten.

Neue Programme sehen den gesetzten Wert sofort. Programme, die schon offen waren, übernehmen ihn nach einem Neustart.

## Konfiguration

Die Datei heißt `set-env.config.yaml` und liegt neben der exe.

```yaml
titel: Arbeitsplatz festlegen
darkmode: false
variable: IGELNAME
spalten: 7
gruppen:
  - name: Labor
    buttons:
      - anzeigename: Labb2plaz
        wert: Igel1
  - name: Büro
    buttons:
      - anzeigename: Eintrag 2
        wert: Igel05
```

| Feld | Bedeutung |
| --- | --- |
| `titel` | Text in der Titelleiste. Ohne Angabe steht dort `Arbeitsplatz festlegen`. |
| `darkmode` | `true` schaltet auf dunkle Farben. Ohne Angabe oder mit `false` bleibt das Fenster hell. |
| `variable` | Name der Benutzervariable. Ohne Angabe heißt sie `IGELNAME`. Erlaubt sind Buchstaben, Ziffern und Unterstriche. Das erste Zeichen ist ein Buchstabe oder ein Unterstrich. |
| `spalten` | Wie viele Buttons nebeneinander stehen. Der nächste Button beginnt eine neue Zeile. Ohne Angabe sind es 5. Erlaubt sind 1 bis 20. |
| `gruppen` | Liste der Arbeitsplatzgruppen. |
| `gruppen.name` | Überschrift der Gruppe. |
| `anzeigename` | Text auf dem Button. |
| `wert` | Wert, der in die Benutzervariable geschrieben wird. |

Statt `gruppen` kann eine einfache Liste `buttons` verwendet werden. Dann gibt es keine Gruppenüberschriften und keine Checkboxen zum Ausblenden.

```yaml
variable: IGELNAME
spalten: 5
buttons:
  - anzeigename: Labor 1
    wert: Igel1
```

Nach einer Änderung an der Konfiguration das Fenster neu öffnen.

## Variable löschen

Oben liegt ein flacher Button über die ganze Breite. Ein Klick entfernt die Benutzervariable vollständig. Das gilt als Arbeitsplatzwechsel: eine konfigurierte Applikation wird beendet oder neu gestartet, wenn vorher ein Wert gesetzt war. Ist die Variable nicht vorhanden, ist dieser Button grün. Die Arbeitsplatz-Buttons sind dann grau.

Der Text steht in `fatclient`. Ohne Angabe heißt der Button `Notebook/Workstation`.

```yaml
fatclient: Notebook/Workstation
```

## Applikation beim Wechsel

Beim Wechsel des Arbeitsplatzes kann eine Applikation hart beendet oder neu gestartet werden. Die Einstellung gilt für alle Arbeitsplätze. Ein erneuter Klick auf den schon gewählten Wert löst sie nicht aus.

Die Variable wird zuerst geschrieben. Danach werden alle laufenden Prozesse mit diesem Exe-Namen beendet. Ein Neustart erhält die neue Variable direkt. Schlägt Beenden oder Start fehl, bleibt der Arbeitsplatz gesetzt und es erscheint eine Meldung.

```yaml
applikation:
  aktion: restart
  name: Client.exe
  startbefehl: '"C:\Program Files\Client\Client.exe"'
  starten: true
```

| Feld | Bedeutung |
| --- | --- |
| `aktion` | `none`, `kill` oder `restart`. Ohne `applikation` oder mit `none` passiert nichts. |
| `name` | Dateiname der Exe, zum Beispiel `Client.exe`. Verglichen wird nur der Dateiname, ohne Pfad. |
| `startbefehl` | Kommandozeile für den Neustart, inklusive Pfad und Argumenten. Pflicht bei `restart`. |
| `starten` | `true` startet diese Kommandozeile auch dann, wenn das Programm gerade nicht läuft. Ohne Angabe oder mit `false` wird nur neu gestartet, wenn es vorher lief. |

Bei `kill` wird `startbefehl` nicht verwendet. `starten` gilt nur für `restart`.

## Gruppen ein- und ausblenden

Neben jeder Gruppe steht die Checkbox **ausblenden**. Ein Haken blendet diese Gruppe sofort aus und merkt sich das für den nächsten Start.

**Alle anzeigen** oben rechts blendet jede Gruppe wieder ein.

Diese Auswahl ist benutzerspezifisch und liegt unter:

`%APPDATA%\SET-ENV\gruppen.yaml`

Beispiel:

```yaml
ausgeblendet:
  - Büro
```

Die Datei wird vom Programm geschrieben. Sie muss nicht von Hand angelegt werden. Wird sie gelöscht, sind beim nächsten Start wieder alle Gruppen sichtbar.

Ausser dieser Datei legt das Programm nichts ab. Es gibt keinen Grafikzwischenspeicher und keinen weiteren Ordner.
