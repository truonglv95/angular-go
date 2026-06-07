export function getElement(elementOrSelector: HTMLElement | string): HTMLElement {
  if (typeof elementOrSelector === 'string') {
    const el = document.querySelector(elementOrSelector);
    if (!el) {
      throw new Error(`Element not found for selector: "${elementOrSelector}"`);
    }
    return el as HTMLElement;
  }
  return elementOrSelector;
}

export function dispatch(elementOrSelector: HTMLElement | string, eventName: string, eventObj?: any): void {
  const element = getElement(elementOrSelector);
  const event = new Event(eventName, { bubbles: true, cancelable: true });
  if (eventObj) {
    Object.assign(event, eventObj);
  }
  defineEventValue(event, 'target', element);
  defineEventValue(event, 'currentTarget', element);
  element.dispatchEvent(event);
}

export function type(elementOrSelector: HTMLElement | string, text: string): void {
  const element = getElement(elementOrSelector) as HTMLInputElement;
  element.focus();
  element.value = text;
  dispatch(element, 'input');
  dispatch(element, 'change');
}

export function clear(elementOrSelector: HTMLElement | string): void {
  const element = getElement(elementOrSelector) as HTMLInputElement;
  element.focus();
  element.value = '';
  dispatch(element, 'input');
  dispatch(element, 'change');
}

export function check(elementOrSelector: HTMLElement | string, checked = true): void {
  const element = getElement(elementOrSelector) as HTMLInputElement;
  element.checked = checked;
  dispatch(element, 'click');
  dispatch(element, 'change');
}

export function select(elementOrSelector: HTMLElement | string, value: string): void {
  const element = getElement(elementOrSelector) as HTMLSelectElement;
  element.value = value;
  dispatch(element, 'change');
}

function defineEventValue(event: Event, key: 'target' | 'currentTarget', value: HTMLElement): void {
  try {
    Object.defineProperty(event, key, {
      configurable: true,
      enumerable: true,
      value
    });
  } catch {
    // Some DOM implementations expose these as non-configurable.
  }
}
