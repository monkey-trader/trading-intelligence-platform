import styles from '@/presentation/position/components/PositionTabs.module.css';

export type PositionTab = 'trades' | 'swing-long';

type PositionTabsProps = Readonly<{
  active: PositionTab;
  onChange: (tab: PositionTab) => void;
}>;

const TABS: readonly Readonly<{ id: PositionTab; label: string }>[] = [
  { id: 'trades', label: 'Trades' },
  { id: 'swing-long', label: 'Swing Long' },
];

/** Reiter "Trades" | "Swing Long". */
export function PositionTabs({ active, onChange }: PositionTabsProps) {
  return (
    <div role="tablist" aria-label="Ansicht" className={styles.tabs}>
      {TABS.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={tab.id === active}
          className={styles.tab}
          onClick={() => onChange(tab.id)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}
