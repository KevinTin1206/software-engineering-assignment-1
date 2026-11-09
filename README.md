# Software Engineering · Aufgabe 1

Zwei eigenständig lauffähige Go-Webanwendungen zur Verwaltung von Ausleihen.

| Aufgabenteil | Projekt | Lokale Adresse |
| --- | --- | --- |
| a+b | [Skiverleih](ski-rental-ab/README.md) | http://127.0.0.1:8080 |
| c | [Fahrradverleih](bike-rental-c/README.md) | http://127.0.0.1:8081 |

Voraussetzung: Go 1.23 oder neuer. Im jeweiligen Projektordner startet `go run .`
die Anwendung. Die Tests laufen dort mit `go test ./...`.

Beide Projekte enthalten ein Dashboard, Ausrüstungs- und Personenverwaltung,
Reservierung, Ausgabe und Rückgabe, Preisstaffeln und einen Ausweisbild-Upload
für fiktive Testdaten. Änderungen werden im Arbeitsspeicher gehalten und beim
Neustart zurückgesetzt.
