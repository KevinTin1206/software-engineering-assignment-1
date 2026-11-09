# Fahrrad-Verleih — Software Engineering, Aufgabe 1 (c)

Ein lauffähiger Prototyp für die Verwaltung eines Fahrrad-Verleihs mit Dashboard,
flexiblen Preisen und Ausweisbild-Upload. Backend: Go; Frontend: HTML/CSS/JavaScript;
Speicher: ausschließlich In-Memory. Keine externen Pakete, kein npm, keine Datenbank.

## Starten

Voraussetzung: **Go 1.23 oder neuer**, moderner Browser.
Im entpackten Projektverzeichnis:

```sh
go run .
```

Im Browser **http://127.0.0.1:8081** öffnen. `Strg+C` beendet den Server.
Nicht `index.html` direkt öffnen: Das Frontend verwendet die Go-API.
Ist Port 8081 belegt, z. B. `go run . -addr 127.0.0.1:8090` verwenden.
Der Server bindet standardmäßig nur an den lokalen Rechner.

```sh
go test ./...
go test -race ./...
go vet ./...
go fmt ./...
go build -o rental .
```

`-race` benötigt eine Go-Installation mit CGO/C-Compiler. Unter Windows heißt
die ausführbare Datei üblicherweise `rental.exe`. Die HTML/JS/CSS-Dateien sind
mit `go:embed` eingebettet. Nach Frontend-Änderungen den Go-Server neu starten.

## Funktionsumfang und Zuordnung

| Aufgabe | Umsetzung |
| --- | --- |
| a | Fahrrad-Bestand und Personen anlegen, bearbeiten, suchen, löschen; Reservierungen; Ausgabe; Rückgabe; Storno; Dashboard und Schnellzugriff |
| b | Tagespreise pro Kategorie, frei editierbare Mindestdauer-Preisstaffeln, Preisvorschau und gespeicherte Preisvereinbarung |
| Zusätzlich vereinbart | Ausweistyp/-nummer und ein JPG/PNG pro Person hochladen, ansehen, ersetzen und löschen |
| c | Dieses eigenständige Fahrradprojekt; Basis aus a+b übernommen und fachlich angepasst |

## Demonstration (ca. 5 Minuten)

1. Starten: Sechs Fahrräder, drei fiktive Personen, eine aktive Ausleihe und eine
   heutige Reservierung sind vorbereitet. Das Dashboard zeigt **4 verfügbar,
   1 ausgeliehen, 6 gesamt, 3 Personen**, zusätzlich 1 heute reserviert.
2. Unter **Personen** eine Person anlegen. Danach **Ausweisbild** öffnen,
   das mitgelieferte **fiktive** `docs/demo-document.png` auswählen und speichern. Wieder öffnen, ansehen,
   ersetzen oder löschen. Ausweistyp/-nummer können ohne Bild gespeichert werden.
3. Unter **Ausrüstung** ein Fahrrad anlegen, z. B. Kategorie City, Rahmengröße 52 cm.
4. **Neue Reservierung**: Person, Beginn heute, Ende heute + 2 Tage wählen,
   danach freie Ausrüstung wählen. Es werden **3 Kalendertage** berechnet.
   City: 18,00 EUR/Tag; ab 3 Tagen 15,00 EUR/Tag → **45,00 EUR**.
5. Reservieren und unter **Reservierungen** auf **Ausgeben** klicken.
   Das Dashboard aktualisiert sich. Während der Ausleihe ist das Fahrrad belegt.
6. Unter **Preise** den City-Tarif ändern. Der schon gespeicherte Vorgang
   behält **45,00 EUR**. Neue Preisvorschauen verwenden den neuen Tarif.
7. **Zurücknehmen** gibt das Fahrrad frei. Eine andere Reservierung kann storniert
   werden. Die historischen Vorgänge bleiben erhalten.
8. Neustart: Alle Änderungen und hochgeladenen Bilder sind weg;
   die vordefinierten Demodaten werden erneut geladen.

## Geschäftsregeln

- Jede Ausrüstung ist ein einzelnes physisches Exemplar mit eigener ID.
- Beginn und Ende sind eingeschlossene Kalendertage: 1.–3. November = 3 Tage.
  Derselbe Tag = 1 Tag. Datumsvergleiche arbeiten mit Datum, nicht mit Uhrzeit;
  die heutige Geschäftszeit folgt der lokalen Zeitzone des Servers.
- Beginn darf beim Anlegen nicht in der Vergangenheit liegen.
- Reservierte und aktive Vorgänge blockieren überlappende Zeiträume.
  Ein Ende am 3. und neuer Beginn am 3. überschneiden sich.
- Ausgabe ist nur zwischen Beginn und Ende einer Reservierung möglich.
  Eine noch nicht zurückgegebene Ausrüstung kann nicht erneut ausgegeben werden,
  auch wenn der vorherige Vorgang überfällig ist. Überfällige physische Ausleihen
  blockieren mindestens bis einschließlich heute; zukünftige Reservierungen
  bleiben möglich, ihre Ausgabe erfordert die tatsächliche vorherige Rückgabe.
- Storno ist nur für Reservierungen möglich; Rückgabe nur für aktive Ausleihen.
  Frühe Rückgabe gibt den verbleibenden Zeitraum frei. Es gibt keine automatische
  Neuberechnung, Rückerstattung oder zusätzliche Verspätungsgebühr.
- Personen und Ausrüstung mit Verleihhistorie werden vor Löschung geschützt.
  Bei Ausrüstung mit Historie bleibt die Kategorie unveränderlich.
- Preise in **ganzzahligen Euro-Cent**, kein Geldrechnen mit Gleitkommazahlen
  im Backend. Tagespreis: 0,01 bis 1.000.000,00 EUR.
- Die Staffel mit der höchsten passenden Mindestdauer gilt **für alle Tage**.
  Beispiel: City 18 EUR, ab 3 Tagen 15 EUR, ab 7 Tagen 13 EUR:
  2 Tage = 36 EUR, 3 Tage = 45 EUR, 7 Tage = 91 EUR.
- Bei Bestätigung übermittelt das Frontend den angezeigten Gesamtpreis.
  Hat sich der Tarif inzwischen geändert, antwortet die API mit 409, zeigt den
  neuen Preis und verlangt einen weiteren Bestätigungsklick. Direkte API-Clients
  können dazu `expectedTotalCents` mitsenden; ohne dieses optionale Feld wird
  der zum Buchungszeitpunkt gültige Preis verwendet.
- Preis, Dauer und Regel werden beim Reservieren gespeichert. Spätere
  Tarifänderungen verändern bestehende Vorgänge nicht.
- Ein optionales Bild je Person: JPG/PNG, max. **5 MiB (5.242.880 Bytes)** und
  **20 Megapixel**. Der Server prüft und dekodiert den tatsächlichen Inhalt;
  Dateiendung und vom Browser behaupteter MIME-Typ reichen nicht aus.
- Alle Daten einschließlich Bilder sind im Arbeitsspeicher. Map und Mutex
  schützen gleichzeitige Requests; Verfügbarkeitsprüfung und Buchung sind atomar.

## Aufbau

```text
main.go                 Start, Konfiguration, fiktive Demodaten
internal/domain/        Datenmodelle, Datum und Preisberechnung
internal/store/         Thread-sicherer In-Memory-Speicher und Geschäftsabläufe
internal/handler/       HTTP-Endpunkte, JSON-/Upload-Validierung
web/                    HTML, JavaScript, CSS, eingebettete Assets
docs/                   Design, Plan und Zuordnung zu den Aufgaben
CLAUDE.md               Unveränderter Text der Lehrvorlage
```

Es gibt bewusst kein generisches Plugin-System und kein DI-Framework.
Structs mit Methoden kapseln die Business-Logik; Handler erhalten den Store
explizit. Das Frontend spricht nur die HTTP-API an.

## HTTP-API

| Methode | Pfad | Zweck |
| --- | --- | --- |
| GET | `/api/config` | Titel und domänenspezifische Beschriftungen |
| GET | `/api/state` | Bestand, Personen, Vorgänge, Tarife, Dashboard, heutiges Datum |
| POST | `/api/items` | Ausrüstung anlegen |
| PUT / DELETE | `/api/items/{id}` | Ausrüstung bearbeiten / löschen |
| POST | `/api/people` | Person anlegen |
| PUT / DELETE | `/api/people/{id}` | Person bearbeiten / löschen |
| POST / GET / DELETE | `/api/people/{id}/document` | Bild hochladen / anzeigen / löschen |
| PUT | `/api/rates` | Kategorie-Tarif vollständig ersetzen |
| GET | `/api/quote?itemId=2&startDate=2026-11-01&endDate=2026-11-03` | Preisvorschau (Daten durch aktuelle/zukünftige ersetzen) |
| POST | `/api/rentals` | Reservieren |
| POST | `/api/rentals/{id}/checkout` | Ausgeben |
| POST | `/api/rentals/{id}/return` | Zurücknehmen |
| POST | `/api/rentals/{id}/cancel` | Stornieren |

JSON-Beispiele:

```json
{"name":"Demo Fahrrad","category":"City","size":"52 cm","details":"Demo"}
```

```json
{"firstName":"Ada","lastName":"Demo","email":"ada@example.invalid","phone":"","documentType":"Demo-Ausweis","documentNumber":"FAKE-100"}
```

```json
{"category":"City","dailyCents":1800,"tiers":[{"minDays":3,"dailyCents":1500},{"minDays":7,"dailyCents":1300}]}
```

```json
{"itemId":2,"personId":1,"startDate":"2026-11-01","endDate":"2026-11-03"}
```

Uploads verwenden `multipart/form-data` mit genau einer Datei im Feld
`document`. Fehlerantworten enthalten `{"error":"..."}`. Statuscodes:
400 Eingabe ungültig, 404 nicht gefunden, 409 Konflikt, 413 zu groß,
415 falscher Content-Type. API-Antworten und Bilder werden nicht gecacht.

## Tests

Die Tests verwenden echte Geschäftsobjekte und `httptest`, keine Mock-Datenbank.
Abgedeckt werden Tagesgrenzen/Schaltjahr, Preisstaffeln, ungültige Eingaben,
Preis-Snapshots und veraltete Preisbestätigungen, inklusive Überschneidungen, Storno, Ausgabe/Rückgabe,
überfällige physische Ausleihen, 32 konkurrierende Buchungen/Ausgaben,
Löschschutz, isolierte Snapshots, HTTP-Abläufe und Ausweisbild-Validierung.

## Git und Abgabe

Für diesen Projektordner wurde ein **eigenes lokales Git-Repository** mit
schrittweisen Commits erstellt (Branch `codex/bike-rental`). Es wurde nichts auf
GitHub veröffentlicht. Das ZIP enthält Quellcode und Tests, aber keine `.git`,
Build-Caches oder plattformspezifischen Binärdateien. Im entpackten ZIP kann
mit `git init` ein neues Repository begonnen werden. Die Entwicklungshistorie
liegt im ursprünglichen lokalen Projektordner.

Die Screenshots nennen zwei ZIPs, liefern aber nicht alle Details der
Vorlesungsdemo. Die in `docs/DESIGN.md` dokumentierten Entscheidungen wurden
vor Implementierung im Chat abgestimmt. Ein öffentliches GitHub-Repository
und die genaue Abgabezeit sind aus den Anhängen nicht sicher bestätigt.

## Grenzen des Prototyps

Kein Login, keine Rollen, keine Zahlungen und keine dauerhafte Speicherung.
Für eine lokale Vorführung mit fiktiven Daten vorgesehen. Kein produktiver
Dienst für echte Ausweise. Der Zusatzpunkt d (Framework/Plugin-Architektur)
ist nicht Bestandteil des vereinbarten a–c-Umfangs.

## 中文快速使用

- 安装 Go 1.23+，终端进入解压后的项目目录，执行 `go run .`。
- 浏览器打开 http://127.0.0.1:8081；停止按 Ctrl+C。
- 导航：Dashboard 总览、Ausrüstung 装备、Personen 人员、Reservierungen 预约和租赁、Preise 价格。
- 新增人员后点击 Ausweisbild 上传虚构证件图；支持预览、替换、删除。
- 新预约先选人员、日期、装备，查看报价后确认；Ausgeben 借出，Zurücknehmen 归还，Stornieren 取消。
- 起止日期均计费；内存存储重启即清空，不要输入真实证件信息。

## Wiederverwendung aus dem Ski-Projekt

`internal/domain`, `internal/store`, `internal/handler` und die Web-Oberfläche
verwenden dasselbe getestete Geschäftsmodell. Wiederverwendbar sind Personen,
Ausweise, inklusive Mietzeiträume, Kollisionsprüfung, Statusübergänge, Preisstaffeln
und Preis-Snapshots. `Item` repräsentiert ein physisches Verleihobjekt.

Fahrradspezifisch sind der Titel, „Rahmengröße“, die Kategorien **City, Trekking,
E-Bike**, die Beispieldaten und ihre Preise (18, 24, 35 EUR pro Tag), sowie
Port 8081. Alle benötigten Dateien liegen in diesem Projekt. Es gibt weder
Importe aus dem Ski-Projekt noch eine externe gemeinsame Bibliothek.
Die bewusste Codekopie hält beide Abgaben unabhängig; das ist ein Prototyp-
Trade-off, keine vorgeschlagene langfristige Paketarchitektur.
