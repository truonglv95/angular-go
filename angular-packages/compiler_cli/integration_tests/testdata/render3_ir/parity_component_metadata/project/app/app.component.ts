import {
  Component,
  Input,
  Output,
  EventEmitter,
  ViewChild,
  ViewChildren,
  ContentChild,
  ContentChildren,
  HostBinding,
  HostListener,
  ChangeDetectionStrategy,
  ViewEncapsulation,
  input,
  model,
  ElementRef,
  QueryList
} from '@angular/core';

@Component({
  selector: 'app-root',
  template: `
    <div>
      Hello Component Metadata Parity!
    </div>
  `,
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  encapsulation: ViewEncapsulation.None,
  preserveWhitespaces: true
})
export class AppComponent {
  @ContentChildren('myContentList') contentList!: QueryList<ElementRef>;

  @ContentChild('myContentRef') contentRef!: ElementRef;

  @Input('aliased') customInput: string = '';

  @Output('aliasedOut') customOutput = new EventEmitter<string>();

  @HostBinding('class.active') isActive = true;

  modelInput = model<boolean>(false);

  @HostListener('click', ['$event'])
  onClick(event: Event) {
    this.standardOutput.emit();
  }

  @Input({ required: true }) requiredInput!: string;

  requiredModelInput = model.required<string>({ alias: 'reqModel' });

  requiredSignalInput = input.required<string>({ alias: 'reqSignal' });

  @HostBinding('attr.role') role = 'button';

  signalInput = input<number>(0);

  @Input() standardInput: string = '';

  @Output() standardOutput = new EventEmitter<void>();

  @ViewChildren('myViewList') viewList!: QueryList<ElementRef>;

  @ViewChild('myViewRef') viewRef!: ElementRef;
}
