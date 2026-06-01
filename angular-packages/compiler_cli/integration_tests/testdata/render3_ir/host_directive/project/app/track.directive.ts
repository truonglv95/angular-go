import {Directive, HostBinding, HostListener, Input, Output} from '@angular/core';

@Directive({
  selector: '[appTrack]',
  standalone: true,
})
export class TrackDirective {
  @Input('appTrack') value = '';
  @Output('selected') selected: unknown = null;
  @HostBinding('class.active') active = true;

  @HostListener('click', ['$event'])
  handleClick(event: Event): void {
    this.selected = event;
  }
}
