import { Component } from '@angular/core';
import { CdkDragDrop, moveItemInArray, CdkDropList, CdkDrag } from '@angular/cdk/drag-drop';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { Store } from '@ngrx/store';
import { map } from 'rxjs/operators';
import { Observable } from 'rxjs';
import { AsyncPipe } from '@angular/common';

// Basic NgRx Store Setup
export const counterReducer = (state = 0, action: any) => {
  switch (action.type) {
    case 'INCREMENT':
      return state + 1;
    case 'DECREMENT':
      return state - 1;
    case 'RESET':
      return 0;
    default:
      return state;
  }
};

@Component({
  selector: 'app-library-test',
  standalone: true,
  imports: [CdkDropList, CdkDrag, MatButtonModule, MatCardModule, AsyncPipe],
  template: `
    <div class="flex flex-col gap-6">
      
      <!-- Angular Material -->
      <mat-card>
        <mat-card-header>
          <mat-card-title>Angular Material Test</mat-card-title>
        </mat-card-header>
        <mat-card-content class="pt-4">
          <p>These buttons are from @angular/material/button</p>
          <div class="flex gap-4 mt-4">
            <button mat-button>Basic</button>
            <button mat-raised-button color="primary">Primary</button>
            <button mat-stroked-button color="accent">Accent</button>
          </div>
        </mat-card-content>
      </mat-card>

      <!-- Angular CDK Drag & Drop -->
      <mat-card>
        <mat-card-header>
          <mat-card-title>Angular CDK Drag & Drop Test</mat-card-title>
        </mat-card-header>
        <mat-card-content class="pt-4">
          <div cdkDropList class="border rounded p-4 bg-gray-50 min-h-[100px]" (cdkDropListDropped)="drop($event)">
            @for (movie of movies; track movie) {
              <div cdkDrag class="p-3 bg-white border mb-2 cursor-move shadow-sm">
                {{movie}}
              </div>
            }
          </div>
        </mat-card-content>
      </mat-card>

      <!-- NgRx Store -->
      <mat-card>
        <mat-card-header>
          <mat-card-title>NgRx Store Test</mat-card-title>
        </mat-card-header>
        <mat-card-content class="pt-4">
          <p class="text-xl font-bold mb-4">Current Count: {{ count$ | async }}</p>
          <div class="flex gap-4">
            <button mat-raised-button color="primary" (click)="increment()">Increment</button>
            <button mat-raised-button color="accent" (click)="decrement()">Decrement</button>
            <button mat-button color="warn" (click)="reset()">Reset</button>
          </div>
        </mat-card-content>
      </mat-card>

    </div>
  `
})
export class LibraryTestComponent {
  movies = [
    'Episode I - The Phantom Menace',
    'Episode II - Attack of the Clones',
    'Episode III - Revenge of the Sith',
    'Episode IV - A New Hope',
    'Episode V - The Empire Strikes Back',
    'Episode VI - Return of the Jedi'
  ];

  count$: Observable<number>;

  constructor(private store: Store<{ count: number }>) {
    this.count$ = store.select('count');
  }

  drop(event: CdkDragDrop<string[]>) {
    moveItemInArray(this.movies, event.previousIndex, event.currentIndex);
  }

  increment() {
    this.store.dispatch({ type: 'INCREMENT' });
  }

  decrement() {
    this.store.dispatch({ type: 'DECREMENT' });
  }

  reset() {
    this.store.dispatch({ type: 'RESET' });
  }
}
