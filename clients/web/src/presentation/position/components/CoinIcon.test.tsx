import { fireEvent, render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { CoinIcon } from '@/presentation/position/components/CoinIcon';

const ICON_URL = 'https://coin-images.coingecko.com/coins/images/26580/large/ONDO.png';

function expectDecorative(img: Element | null) {
  expect(img).not.toBeNull();
  expect(img).toHaveAttribute('alt', '');
  expect(
    img?.getAttribute('aria-hidden') === 'true' || img?.closest('[aria-hidden="true"]'),
  ).toBeTruthy();
}

describe('CoinIcon', () => {
  it('zeigt bei vorhandener iconUrl ein dekoratives img mit dieser URL', () => {
    const { container } = render(
      <CoinIcon baseAsset="ONDO" displayName="ONDO" iconUrl={ICON_URL} />,
    );
    const img = container.querySelector('img');
    expectDecorative(img);
    expect(img).toHaveAttribute('src', ICON_URL);
  });

  it('XAU nutzt das lokale Gold-Icon, auch wenn eine iconUrl geliefert wird', () => {
    const { container } = render(
      <CoinIcon baseAsset="XAU" displayName="Gold" iconUrl={ICON_URL} />,
    );
    const img = container.querySelector('img');
    expectDecorative(img);
    expect(img?.getAttribute('src')?.toLowerCase()).toContain('gold');
    expect(img?.getAttribute('src')).not.toBe(ICON_URL);
  });

  it('XAG nutzt das lokale Silber-Icon ohne iconUrl', () => {
    const { container } = render(<CoinIcon baseAsset="XAG" displayName="Silber" iconUrl={null} />);
    const img = container.querySelector('img');
    expectDecorative(img);
    expect(img?.getAttribute('src')?.toLowerCase()).toContain('silver');
  });

  it.each([
    ['NICKEL', 'Nickel', 'N'],
    ['ENA', 'ena', 'E'],
    ['1000PEPE', '1000PEPE', '1'],
  ])('ohne iconUrl zeigt %s den Fallback mit Buchstabe %s', (baseAsset, displayName, letter) => {
    const { container } = render(
      <CoinIcon baseAsset={baseAsset} displayName={displayName} iconUrl={null} />,
    );
    expect(container.querySelector('img')).toBeNull();
    expect(container).toHaveTextContent(new RegExp(`^${letter.toUpperCase()}$`));
    expect(container.querySelector('[aria-hidden="true"]')).not.toBeNull();
  });

  it('wechselt bei Ladefehler zum Fallback mit erstem Buchstaben des Anzeigenamens', () => {
    const { container } = render(
      <CoinIcon baseAsset="ONDO" displayName="ONDO" iconUrl={ICON_URL} />,
    );
    const img = container.querySelector('img');
    expect(img).not.toBeNull();
    fireEvent.error(img as HTMLImageElement);
    expect(container.querySelector('img')).toBeNull();
    expect(container).toHaveTextContent(/^O$/);
  });

  it('wechselt auch beim lokalen Metall-Icon bei Ladefehler zum Fallback', () => {
    const { container } = render(<CoinIcon baseAsset="XAU" displayName="Gold" iconUrl={null} />);
    fireEvent.error(container.querySelector('img') as HTMLImageElement);
    expect(container.querySelector('img')).toBeNull();
    expect(container).toHaveTextContent(/^G$/);
  });
});
