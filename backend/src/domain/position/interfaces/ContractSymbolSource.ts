/** Port: liefert alle handelbaren Kontrakt-Symbole der Börse (für die einmalige Suche nach alten Symbolen). */
export type ContractSymbolSource = {
  listContractSymbols(): Promise<string[]>;
};
