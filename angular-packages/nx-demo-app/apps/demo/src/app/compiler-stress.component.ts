import {
  Attribute,
  ChangeDetectionStrategy,
  Component,
  ContentChild,
  Directive,
  ElementRef,
  HostBinding,
  HostListener,
  Inject,
  InjectionToken,
  Input,
  Optional,
  Pipe,
  PipeTransform,
  Self,
  SkipSelf,
  TemplateRef,
  booleanAttribute,
  computed,
  effect,
  inject,
  input,
  model,
  numberAttribute,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormArray, FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';

export const STRESS_TOKEN = new InjectionToken<StressConfig>('STRESS_TOKEN', {
  providedIn: 'root',
  factory: () => ({ label: 'root-token', multiplier: 3 }),
});

export interface StressConfig {
  label: string;
  multiplier: number;
}

type StressKind = 'alpha' | 'beta' | 'gamma';

interface StressItem {
  id: number;
  kind: StressKind;
  title: string;
  active: boolean;
  score: number;
  meta?: {
    owner?: string;
    tags: string[];
  };
}

@Directive({
  selector: '[appStressHighlight]',
  standalone: true,
  host: {
    '[class.is-highlighted]': 'enabled',
    '[attr.data-highlight-tone]': 'tone',
  },
})
export class StressHighlightDirective {
  @Input({ alias: 'appStressHighlight', transform: booleanAttribute })
  enabled = false;

  @Input({ transform: numberAttribute })
  weight = 1;

  @HostBinding('style.--stress-weight')
  get stressWeight() {
    return `${this.weight}`;
  }

  tone = 'teal';
  private readonly el = inject<ElementRef<HTMLElement>>(ElementRef);

  @HostListener('mouseenter')
  markWarm() {
    this.tone = this.el.nativeElement.dataset['tone'] || 'teal';
  }
}

@Pipe({
  name: 'stressScore',
  standalone: true,
})
export class StressScorePipe implements PipeTransform {
  transform(value: number | null | undefined, multiplier = 1, fallback = 'n/a'): string {
    if (value == null || Number.isNaN(value)) {
      return fallback;
    }

    return `${Math.round(value * multiplier)} pts`;
  }
}

@Component({
  selector: 'app-stress-card',
  standalone: true,
  imports: [CommonModule, StressHighlightDirective, StressScorePipe],
  hostDirectives: [StressHighlightDirective],
  template: `
    <article
      class="stress-card"
      [ngClass]="{ 'stress-card--active': item().active, 'stress-card--quiet': !item().active }"
      [appStressHighlight]="item().active"
      [weight]="item().score"
      [attr.data-kind]="item().kind"
    >
      <header>
        <ng-content select="[card-kicker]"></ng-content>
        <h3>{{ item().title }}</h3>
      </header>

      <p>{{ formatScore(item().score, multiplier(), 'missing') }}</p>

      @if (detailsTemplate()) {
        <ng-container
          [ngTemplateOutlet]="detailsTemplate()"
          [ngTemplateOutletContext]="{ $implicit: item(), kind: item().kind }"
        />
      } @else {
        <small>No projected template</small>
      }

      <button type="button" (click)="selected.emit(item().id)">Select</button>
    </article>
  `,
})
export class StressCardComponent {
  item = input.required<StressItem>();
  multiplier = input(1, { transform: numberAttribute });
  selected = output<number>();

  @ContentChild(TemplateRef)
  projected?: TemplateRef<{ $implicit: StressItem; kind: StressKind }>;

  detailsTemplate = computed(() => this.projected ?? null);

  formatScore(value: number | null | undefined, multiplier = 1, fallback = 'n/a'): string {
    if (value == null || Number.isNaN(value)) {
      return fallback;
    }

    return `${Math.round(value * multiplier)} pts`;
  }
}

@Component({
  selector: 'app-model-counter',
  standalone: true,
  template: `
    <button type="button" (click)="count.update((value) => value - step())">-</button>
    <output>{{ count() }}</output>
    <button type="button" (click)="count.update((value) => value + step())">+</button>
  `,
})
export class ModelCounterComponent {
  count = model(0);
  step = input(1, { transform: numberAttribute });
}

@Component({
  selector: 'app-compiler-stress',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, StressCardComponent, ModelCounterComponent, StressScorePipe],
  providers: [
    {
      provide: STRESS_TOKEN,
      useValue: { label: 'component-token', multiplier: 5 } satisfies StressConfig,
    },
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="stress-shell">
      <header>
        <p>{{ upper(config.label) }}</p>
        <h2>Compiler stress route</h2>
        <span>attr={{ modeAttr || 'none' }} optional={{ optionalConfig?.label || 'none' }}</span>
      </header>

      @let summary = summaryText();
      <p class="summary">{{ summary }}</p>

      <form [formGroup]="form" class="stress-form">
        <label>
          Filter
          <input formControlName="filter" />
        </label>

        <label>
          Minimum score
          <input type="number" formControlName="minimumScore" />
        </label>

        <div formArrayName="toggles">
          @for (toggle of toggles.controls; track toggle; let idx = $index) {
            <label>
              <input type="checkbox" [formControlName]="idx" />
              Toggle {{ idx + 1 }}
            </label>
          }
        </div>
      </form>

      <app-model-counter [(count)]="counter" [step]="2" />

      @if (vm(); as vm) {
        <section class="stress-grid">
          @for (item of vm.items; track item.id; let idx = $index; let odd = $odd) {
            <app-stress-card
              [item]="item"
              [multiplier]="config.multiplier"
              (selected)="selectItem($event)"
            >
              <span card-kicker>#{{ idx + 1 }} {{ odd ? 'odd' : 'even' }}</span>
              <ng-template let-card let-kind="kind">
                @switch (kind) {
                  @case ('alpha') {
                    <strong>Alpha owner: {{ card.meta?.owner ?? 'unknown' }}</strong>
                  }
                  @case ('beta') {
                    <strong>Beta tags: {{ card.meta?.tags?.join(', ') || 'none' }}</strong>
                  }
                  @default {
                    <strong>Gamma json: {{ json(card.meta) }}</strong>
                  }
                }
              </ng-template>
            </app-stress-card>
          } @empty {
            <p>No stress items match the current filter.</p>
          }
        </section>
      }

      <button #deferTrigger type="button">Hydrate deferred block</button>
      @defer (on interaction(deferTrigger); prefetch on idle) {
        <aside class="deferred">
          <p>Deferred selected id: {{ selectedId() ?? 'none' }}</p>
          <p>Today: {{ todayText }}</p>
          <p>Score preview: {{ formatScore(selectedScore(), counter(), 'zero') }}</p>
        </aside>
      } @placeholder {
        <p class="placeholder">Deferred block is waiting for interaction.</p>
      } @loading (minimum 80ms) {
        <p class="placeholder">Loading deferred block...</p>
      }
    </section>
  `,
  styles: [`
    :host {
      display: block;
    }

    .stress-shell {
      display: grid;
      gap: 16px;
      padding: 20px;
      border: 1px solid #cdd7e1;
      border-radius: 10px;
      background: #ffffff;
    }

    header {
      display: grid;
      gap: 4px;
    }

    header p,
    header h2 {
      margin: 0;
    }

    .summary {
      margin: 0;
      color: #3f4c5f;
      font-weight: 700;
    }

    .stress-form,
    .stress-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
      gap: 12px;
    }

    .stress-form label {
      display: grid;
      gap: 5px;
      font-weight: 700;
    }

    app-model-counter {
      display: inline-grid;
      grid-template-columns: 36px minmax(44px, auto) 36px;
      align-items: center;
      gap: 8px;
      width: max-content;
    }

    .stress-card {
      display: grid;
      gap: 10px;
      min-height: 180px;
      padding: 14px;
      border: 1px solid #d5dde8;
      border-radius: 8px;
      background: #f8fafc;
    }

    .stress-card--active {
      border-color: #0f9f8f;
      box-shadow: inset 4px 0 0 #0f9f8f;
    }

    .stress-card--quiet {
      opacity: 0.78;
    }

    .is-highlighted {
      outline: calc(var(--stress-weight) * 1px) solid rgba(15, 159, 143, 0.18);
    }

    .deferred,
    .placeholder {
      margin: 0;
      padding: 12px;
      border-radius: 8px;
      background: #eef7ff;
    }
  `],
})
export class CompilerStressComponent {
  readonly config = inject(STRESS_TOKEN);
  readonly fb = inject(FormBuilder);

  readonly items = signal<StressItem[]>([
    { id: 1, kind: 'alpha', title: 'Required input and projection', active: true, score: 13, meta: { owner: 'Ada', tags: ['signal', 'pipe'] } },
    { id: 2, kind: 'beta', title: 'Control flow and forms', active: false, score: 21, meta: { tags: ['form-array', 'switch'] } },
    { id: 3, kind: 'gamma', title: 'Deferred and async pipe', active: true, score: 8 },
  ]);

  readonly selectedId = signal<number | null>(null);
  readonly counter = signal(2);
  readonly today = new Date('2026-06-13T00:00:00+07:00');
  readonly todayText = new Intl.DateTimeFormat('en-US', { dateStyle: 'full' }).format(this.today);
  readonly headerRef = viewChild<ElementRef<HTMLElement>>('deferTrigger');
  readonly formValue = signal({ filter: '', minimumScore: 0 });

  readonly form = this.fb.nonNullable.group({
    filter: ['', [Validators.maxLength(30)]],
    minimumScore: [0, [Validators.min(0)]],
    toggles: this.fb.array([true, false, true]),
  });

  readonly vm = computed(() => {
    const filter = this.formValue().filter.trim().toLowerCase();
    const minimumScore = this.formValue().minimumScore;
    return {
      items: this.items().filter((item) =>
        item.score >= minimumScore &&
        (!filter || item.title.toLowerCase().includes(filter) || item.kind.includes(filter))
      ),
    };
  });

  readonly selectedScore = computed(() => {
    const selectedId = this.selectedId();
    return this.items().find((item) => item.id === selectedId)?.score ?? null;
  });

  readonly summaryText = computed(() => {
    const activeCount = this.items().filter((item) => item.active).length;
    return `${activeCount}/${this.items().length} active, counter=${this.counter()}`;
  });

  constructor(
    @Attribute('data-mode') readonly modeAttr: string | null,
    @Optional() @Inject(STRESS_TOKEN) readonly optionalConfig: StressConfig | null,
    @Optional() @Self() @Inject(STRESS_TOKEN) readonly selfConfig: StressConfig | null,
    @Optional() @SkipSelf() @Inject(STRESS_TOKEN) readonly parentConfig: StressConfig | null,
  ) {
    this.form.valueChanges.subscribe((value) => {
      this.formValue.set({
        filter: value.filter ?? '',
        minimumScore: value.minimumScore ?? 0,
      });
    });
  }

  get toggles(): FormArray {
    return this.form.controls.toggles;
  }

  selectItem(id: number) {
    this.selectedId.set(id);
    this.items.update((items) =>
      items.map((item) => item.id === id ? { ...item, active: !item.active } : item)
    );
  }

  upper(value: string): string {
    return value.toUpperCase();
  }

  json(value: unknown): string {
    return JSON.stringify(value ?? null);
  }

  formatScore(value: number | null | undefined, multiplier = 1, fallback = 'n/a'): string {
    if (value == null || Number.isNaN(value)) {
      return fallback;
    }

    return `${Math.round(value * multiplier)} pts`;
  }
}
