import { useState } from 'react';
import goldIconUrl from '@/assets/commodities/gold.svg?no-inline';
import silverIconUrl from '@/assets/commodities/silver.svg?no-inline';
import styles from '@/presentation/position/components/OpenPositionsTable.module.css';

type CoinIconProps = Readonly<{
  baseAsset: string;
  displayName: string;
  iconUrl: string | null;
}>;

const LOCAL_ICONS: Readonly<Record<string, string>> = { XAU: goldIconUrl, XAG: silverIconUrl };

/** Dekoratives Coin-Icon; ohne URL oder bei Ladefehler zeigt es den Anfangsbuchstaben. */
export function CoinIcon({ baseAsset, displayName, iconUrl }: CoinIconProps) {
  const [hasFailed, setHasFailed] = useState(false);
  const source = LOCAL_ICONS[baseAsset] ?? iconUrl;
  if (source === null || source === undefined || hasFailed) {
    return (
      <span aria-hidden="true" className={styles.coinFallback}>
        {displayName.charAt(0).toUpperCase()}
      </span>
    );
  }
  return (
    <img
      aria-hidden="true"
      alt=""
      src={source}
      className={styles.coinIcon}
      onError={() => setHasFailed(true)}
    />
  );
}
