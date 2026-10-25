/** Erklärung der Spalte mit BE und TP1 für den Hinweisdialog. */
export function BreakEvenTargetInfo() {
  return (
    <>
      <p>
        Die Spalte rechnet mit dem <strong>ursprünglichen Risiko (R)</strong>: dem Abstand vom
        Einstieg zum Stop-Loss, den du beim Eröffnen gesetzt hast. Ein späteres Nachziehen des SL
        ändert R nicht.
      </p>
      <ul>
        <li>
          <strong>BE (ab 2R)</strong>: Erreicht der Kurs den Preis bei 2R, ziehst du bei BingX den
          SL auf Break-even, also auf den Einstieg. Danach kann die Position nicht mehr im Verlust
          enden, abgesehen von Gebühren.
        </li>
        <li>
          <strong>TP1 (bei 3R)</strong>: der Preis bei 3R, also ein Chance-Risiko-Verhältnis von
          1:3. Als Richtwert nimmst du dort etwa 33 % der Position raus.
        </li>
      </ul>
      <p>
        Bei Long liegen alle Preise über dem Einstieg, bei Short darunter. Sobald 2R erreicht ist
        und der SL noch nicht auf Break-even steht, erscheint in der Zeile der Hinweis „BE ziehen“.
      </p>
      <p>
        Steht dort „–“, ist das Risiko unbekannt, weil der ursprüngliche SL fehlt. Mit „SL
        eintragen“ kannst du ihn nachtragen.
      </p>
    </>
  );
}
