/** Erklärung der Spalte "TPs (je 33 %)" für den Hinweisdialog. */
export function BreakEvenTargetInfo() {
  return (
    <>
      <p>
        Die Spalte rechnet mit dem <strong>ursprünglichen Risiko (1R)</strong>: dem Abstand vom
        Einstieg zum Stop-Loss, den du beim Eröffnen gesetzt hast. Ein späteres Nachziehen des SL
        ändert 1R nicht.
      </p>
      <p>
        Du nimmst in der Regel <strong>drei Take-Profits mit je 33 %</strong> der Position:
      </p>
      <ul>
        <li>
          <strong>TP1 (1R)</strong>: der Preis, der 1R vom Einstieg entfernt liegt.
        </li>
        <li>
          <strong>BE (2R)</strong>: der Preis bei 2R. Er ist der{' '}
          <strong>Break-even-Auslöser</strong>: Erreicht der Kurs ihn, ziehst du bei BingX den SL
          auf den Einstieg. Danach kann die Position nicht mehr im Verlust enden, abgesehen von
          Gebühren.
        </li>
        <li>
          <strong>TP3 (3R)</strong>: der Preis bei 3R, also ein Chance-Risiko-Verhältnis von 1:3.
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
