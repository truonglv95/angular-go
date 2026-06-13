import { Component } from '@angular/core';

@Component({
  selector: 'demo-module-dashboard-widget',
  standalone: false,
  template: `
    <section class="widget-panel widget-panel--module">
      <p class="widget-kicker">NgModule widget</p>
      <h2>Inventory pulse</h2>
      <p>Loaded through a lazy feature module with RouterModule.forChild.</p>
      <dl>
        <div>
          <dt>Open tasks</dt>
          <dd>18</dd>
        </div>
        <div>
          <dt>Restock risk</dt>
          <dd>3 zones</dd>
        </div>
      </dl>
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

    .widget-panel--module {
      border-left: 5px solid #0f9f8f;
    }

    .widget-kicker {
      margin: 0 0 6px;
      color: #0f766e;
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

    dl {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 10px;
      margin: 0;
    }

    dt {
      color: #5c667a;
      font-size: 0.8rem;
    }

    dd {
      margin: 2px 0 0;
      font-weight: 800;
    }
  `]
})
export class ModuleDashboardComponent {}
