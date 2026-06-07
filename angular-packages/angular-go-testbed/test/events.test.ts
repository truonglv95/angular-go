import { describe, expect, it } from 'vitest';
import { check, clear, dispatch, select, type } from '../src/events.js';

describe('event helpers', () => {
  it('dispatches events with target and currentTarget', () => {
    const button = document.createElement('button');
    let receivedTarget: EventTarget | null = null;
    let receivedCurrentTarget: EventTarget | null = null;
    button.addEventListener('click', (event) => {
      receivedTarget = event.target;
      receivedCurrentTarget = event.currentTarget;
    });

    dispatch(button, 'click');

    expect(receivedTarget).toBe(button);
    expect(receivedCurrentTarget).toBe(button);
  });

  it('types and clears input values using input/change events', () => {
    const input = document.createElement('input');
    const events: string[] = [];
    input.addEventListener('input', () => events.push('input'));
    input.addEventListener('change', () => events.push('change'));

    type(input, 'hello');
    expect(input.value).toBe('hello');
    expect(events).toEqual(['input', 'change']);

    clear(input);
    expect(input.value).toBe('');
    expect(events).toEqual(['input', 'change', 'input', 'change']);
  });

  it('checks inputs and selects options', () => {
    const checkbox = document.createElement('input');
    checkbox.type = 'checkbox';
    check(checkbox, true);
    expect(checkbox.checked).toBe(true);

    const selectEl = document.createElement('select');
    const option = document.createElement('option');
    option.value = 'a';
    selectEl.appendChild(option);
    select(selectEl, 'a');
    expect(selectEl.value).toBe('a');
  });
});

