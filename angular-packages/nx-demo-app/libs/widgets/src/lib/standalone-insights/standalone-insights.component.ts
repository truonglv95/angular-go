import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'demo-standalone-insights-widget',
  standalone: true,
  imports: [CommonModule],
  template: `
    <section class="widget-panel widget-panel--standalone">
      <p class="widget-kicker">Standalone widget</p>
      <h2>Conversion watch</h2>
      <p>Loaded directly with route loadComponent.</p>
      <ul>
        <li *ngFor="let item of signals">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
        </li>
      </ul>
    </section>
  `,
  styles: [`
    .widget-panel {
      border: 1px solid #d6dde8;
      border-radius: 10px;
      padding: 18px;
      background: #ffffff;
      color: #172033;
      box-shadow: 0 10px 22px rgba(23, 32, 51, 0.08);
    }

    .widget-panel--standalone {
      border-left: 5px solid #7b61ff;
    }

    .widget-kicker {
      margin: 0 0 6px;
      color: #5b43d6;
      font-weight: 700;
      font-size: 0.8rem;
      text-transform: uppercase;
    }

    h2 {
      margin: 0 0 6px;
      font-size: 1.2rem;
    }

    p {
      margin: 0 0 14px;
    }

    ul {
      display: grid;
      gap: 8px;
      padding: 0;
      margin: 0;
      list-style: none;
    }

    li {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      padding: 10px 12px;
      border-radius: 8px;
      background: #f6f7fb;
    }
  `]
})
export class StandaloneInsightsComponent {
  protected readonly signals = [
    { label: 'Checkout health', value: '98%' },
    { label: 'Trial starts', value: '+12' },
    { label: 'Queue time', value: '4m' }
  ];
}
