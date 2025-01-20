Umsatzsteuervoranmeldung XML für JES
====================================

Das Elster-Portal bietet die Möglichkeit, die UStVA per [XML einzureichen](https://www.elster.de/eportal/helpGlobal?themaGlobal=ustva_upload). Das erspart einem das Abtippen von Nummern und verhindert so Erfassungs- und Kopierfehler.

[JES](https://www.jes-eur.de/) ist eine kleine EÜR-Verwaltungssoftware, die für einfache Selbständigkeiten alles nötige anbietet. 

Leider bietet es (derzeit) noch keinen XML-Export für die UStVA an. Dieses Tool überbrückt diese Lücke, indem es die JES-Datei einliest und die XML-Datei erzeugt.

### Verwendung

```
jesva jes-datei.eux monat > ustva_monat.xml
```

`monat` kann dabei ein Monat (1-12) oder ein Quartal (Q1-Q4) sein.

**Wichtig**: Für die UStVA werden Daten benötigt, die im JES nicht vorliegen. Diese müssen in einer Datei `config.json` im aktuellen Verzeichnis abgelegt sein. Für Details siehe die [config.example.json](./config.example.json).

### Installation

`go install github.com/Necoro/jesva@latest`

Alternativ:
```  
git clone github.com/Necoro/jesva
cd jesva  
go build
```