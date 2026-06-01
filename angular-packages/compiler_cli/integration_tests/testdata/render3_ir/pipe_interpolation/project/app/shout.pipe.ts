import {Pipe} from '@angular/core';

@Pipe({
  name: 'shout',
  standalone: true,
})
export class ShoutPipe {
  transform(value: string): string {
    return value.toUpperCase();
  }
}
