Umsatzsteuervoranmeldung XML für JES
====================================

Das Elster-Portal bietet die Möglichkeit, die UStVA per [XML einzureichen](https://www.elster.de/eportal/helpGlobal?themaGlobal=ustva_upload). Das erspart einem das Abtippen von Nummern und verhindert so Erfassungs- und Kopierfehler.

[JES](https://www.jes-eur.de/) ist eine kleine EÜR-Verwaltungssoftware, die für einfache Selbständigkeiten alles nötige anbietet. 

Leider bietet es (derzeit) noch keinen XML-Export für die UStVA an. Dieses Tool überbrückt diese Lücke, indem es die JES-Datei einliest und die XML-Datei erzeugt.

### Verwendung

```
jesva jes-datei.eux zeitraum > ustva_monat.xml
```

`zeitraum` kennt dabei mehrere Formate:
* Monat (1-12)
* Quartal (Q1-Q4)
* Monatszeitraum (`start`-`ende`, z.B. `3-5`). **NB**: Das wird sehr selten gebraucht werden und hat auch in den
UStVA-Zeiträumen keine Entsprechung. In der UStVA angedruckt wird `ende`. 

**Wichtig**: Für die UStVA werden Daten benötigt, die im JES nicht vorliegen. Diese müssen in einer Datei `config.json` 
oder `jesva.json` im aktuellen Verzeichnis abgelegt sein. Für Details siehe die [config.example.json](./config.example.json).

### Installation

`go install github.com/Necoro/jesva@latest`

Alternativ:
```  
git clone https://github.com/Necoro/jesva
cd jesva  
go build
```