import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { AssetIcon } from '@/domain/asset/AssetIcon';
import type { AssetIconsGateway } from '@/domain/asset/AssetIconsGateway';
import { useAssetIcons } from '@/presentation/position/hooks/useAssetIcons';

const ondo: AssetIcon = {
  symbol: 'ONDO-USDT',
  baseAsset: 'ONDO',
  displayName: 'ONDO',
  iconUrl: 'https://coin-images.coingecko.com/coins/images/1/large/ondo.png',
};
const gold: AssetIcon = {
  symbol: 'NCCOGOLD2USD-USDT',
  baseAsset: 'XAU',
  displayName: 'Gold',
  iconUrl: null,
};

function gatewayReturning(icons: readonly AssetIcon[]) {
  const fetchAssetIcons = vi.fn<AssetIconsGateway['fetchAssetIcons']>().mockResolvedValue(icons);
  const gateway: AssetIconsGateway = { fetchAssetIcons };
  return { gateway, fetchAssetIcons };
}

describe('useAssetIcons', () => {
  it('ruft einmal für alle Symbole ab und liefert sie pro Symbol', async () => {
    const { gateway, fetchAssetIcons } = gatewayReturning([ondo, gold]);

    const { result } = renderHook(() => useAssetIcons(gateway, ['ONDO-USDT', 'NCCOGOLD2USD-USDT']));

    expect(result.current.get('ONDO-USDT')).toBeUndefined();
    await waitFor(() => expect(result.current.get('ONDO-USDT')).toEqual(ondo));
    expect(result.current.get('NCCOGOLD2USD-USDT')).toEqual(gold);
    expect(result.current.get('UNBEKANNT-USDT')).toBeUndefined();
    expect(fetchAssetIcons).toHaveBeenCalledTimes(1);
    expect(fetchAssetIcons.mock.calls[0]?.[0]).toEqual(['ONDO-USDT', 'NCCOGOLD2USD-USDT']);
  });

  it('ruft bei leerer Liste nicht ab', async () => {
    const { gateway, fetchAssetIcons } = gatewayReturning([]);

    const { result } = renderHook(() => useAssetIcons(gateway, []));

    await Promise.resolve();
    expect(fetchAssetIcons).not.toHaveBeenCalled();
    expect(result.current.get('ONDO-USDT')).toBeUndefined();
  });

  it('ruft nicht erneut ab, wenn nur eine neue Array-Instanz mit gleichem Inhalt kommt', async () => {
    const { gateway, fetchAssetIcons } = gatewayReturning([ondo]);

    const { result, rerender } = renderHook(({ symbols }) => useAssetIcons(gateway, symbols), {
      initialProps: { symbols: ['ONDO-USDT'] },
    });
    await waitFor(() => expect(result.current.get('ONDO-USDT')).toEqual(ondo));
    rerender({ symbols: ['ONDO-USDT'] });
    rerender({ symbols: ['ONDO-USDT'] });

    expect(fetchAssetIcons).toHaveBeenCalledTimes(1);
  });

  it('dedupliziert Symbole und ignoriert die Reihenfolge', async () => {
    const { gateway, fetchAssetIcons } = gatewayReturning([ondo, gold]);

    const { result, rerender } = renderHook(({ symbols }) => useAssetIcons(gateway, symbols), {
      initialProps: { symbols: ['ONDO-USDT', 'NCCOGOLD2USD-USDT', 'ONDO-USDT'] },
    });
    await waitFor(() => expect(result.current.get('ONDO-USDT')).toEqual(ondo));
    rerender({ symbols: ['NCCOGOLD2USD-USDT', 'ONDO-USDT'] });

    expect(fetchAssetIcons).toHaveBeenCalledTimes(1);
    expect(new Set(fetchAssetIcons.mock.calls[0]?.[0])).toEqual(
      new Set(['ONDO-USDT', 'NCCOGOLD2USD-USDT']),
    );
    expect(fetchAssetIcons.mock.calls[0]?.[0]).toHaveLength(2);
  });

  it('ruft erneut ab, wenn ein neues Symbol hinzukommt', async () => {
    const { gateway, fetchAssetIcons } = gatewayReturning([ondo, gold]);

    const { rerender } = renderHook(({ symbols }) => useAssetIcons(gateway, symbols), {
      initialProps: { symbols: ['ONDO-USDT'] },
    });
    await waitFor(() => expect(fetchAssetIcons).toHaveBeenCalledTimes(1));
    rerender({ symbols: ['ONDO-USDT', 'NCCOGOLD2USD-USDT'] });

    await waitFor(() => expect(fetchAssetIcons).toHaveBeenCalledTimes(2));
  });

  it('liefert bei leerem Ergebnis (Fehler im Gateway) für alle Symbole undefined', async () => {
    const { gateway, fetchAssetIcons } = gatewayReturning([]);

    const { result } = renderHook(() => useAssetIcons(gateway, ['ONDO-USDT']));

    await waitFor(() => expect(fetchAssetIcons).toHaveBeenCalledTimes(1));
    expect(result.current.get('ONDO-USDT')).toBeUndefined();
  });

  it('wirft nicht, wenn das Gateway unerwartet ablehnt', async () => {
    const fetchAssetIcons = vi
      .fn<AssetIconsGateway['fetchAssetIcons']>()
      .mockRejectedValue(new Error('kaputt'));

    const { result } = renderHook(() => useAssetIcons({ fetchAssetIcons }, ['ONDO-USDT']));

    await waitFor(() => expect(fetchAssetIcons).toHaveBeenCalledTimes(1));
    expect(result.current.get('ONDO-USDT')).toBeUndefined();
  });

  it('ohne Gateway kein Abruf und kein Fehler', () => {
    const { result } = renderHook(() => useAssetIcons(undefined, ['ONDO-USDT']));
    expect(result.current.get('ONDO-USDT')).toBeUndefined();
  });
});
