import { Component, ViewChild, TemplateRef } from '@angular/core';
import { RouterModule } from '@angular/router';
import { ReactiveFormsModule, FormControl } from '@angular/forms';
import { CommonModule, NgTemplateOutlet } from '@angular/common';
import { NxWelcome } from '@demo/nx-welcome';
import { IcuDemoComponent } from './icu-demo.component';
import { CustomInputComponent } from './custom-input.component';
import { BaseComponent } from './base.component';

import { ChildComponent } from './child.component';

import { MyModule } from './my-module.module';
import { ModuleA } from './module-a.module';
import { NonStandaloneComponent } from './non-standalone.component';

@Component({
  imports: [NxWelcome, RouterModule, IcuDemoComponent, ReactiveFormsModule, CustomInputComponent, ChildComponent, CommonModule, NgTemplateOutlet, MyModule, ModuleA],
  selector: 'app-root',
  templateUrl: './app.html',
  styleUrl: './app.scss',
})
export class App extends BaseComponent {
  protected title = 'demo';
  
  dateControl = new FormControl('2024-01-01');

  setToToday() {
    const today = new Date();
    const yyyy = today.getFullYear();
    const mm = String(today.getMonth() + 1).padStart(2, '0');
    const dd = String(today.getDate()).padStart(2, '0');
    this.dateControl.setValue(`${yyyy}-${mm}-${dd}`);
  }
}
