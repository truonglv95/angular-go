import { ComplexFeatureComponent } from "./complex-feature";
import { LibraryTestComponent } from "./library-test";

import { Component, signal } from '@angular/core';
import { NgIf, NgFor } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterOutlet, RouterLink } from '@angular/router';
import { Menu } from 'primeng/menu';
import { Button } from 'primeng/button';
import { Ripple } from 'primeng/ripple';
import { MenuItem, TreeNode } from 'primeng/api';
import { InputText } from 'primeng/inputtext';
import { TableModule } from 'primeng/table';
import { Checkbox } from 'primeng/checkbox';
import { RadioButton } from 'primeng/radiobutton';
import { Select } from 'primeng/select';
import { ToggleButton } from 'primeng/togglebutton';
import { AccordionModule } from 'primeng/accordion';
import { Dialog } from 'primeng/dialog';
import { TabsModule } from 'primeng/tabs';
import { Slider } from 'primeng/slider';
import { ProgressBar } from 'primeng/progressbar';
import { TreeModule } from 'primeng/tree';
import { Rating } from 'primeng/rating';
import { ToggleSwitch } from 'primeng/toggleswitch';
import { DatePicker } from 'primeng/datepicker';
import { Badge } from 'primeng/badge';
import { Avatar } from 'primeng/avatar';
import { Tag } from 'primeng/tag';
import { Card } from 'primeng/card';
import { Tooltip } from 'primeng/tooltip';
import { Carousel } from 'primeng/carousel';
import { PickList } from 'primeng/picklist';
import { OrderList } from 'primeng/orderlist';
import { TreeTableModule } from 'primeng/treetable';
import { Panel } from 'primeng/panel';
import { Fieldset } from 'primeng/fieldset';
import { Splitter } from 'primeng/splitter';
import { Knob } from 'primeng/knob';
interface Product {
  code: string;
  name: string;
  category: string;
  quantity: number;
}

type DemoView = 'button' | 'input' | 'table' | 'form' | 'accordion' | 'dialog' | 'tabs' | 'misc' | 'tree' | 'extra' | 'advanced' | 'defer' | 'knob' | 'syntax' | 'complex' | 'library';

@Component({
  selector: 'app-root',
  imports: [
    RouterOutlet, RouterLink, NgIf, FormsModule, Menu, Button, Ripple, InputText, TableModule, 
    Checkbox, RadioButton, Select, ToggleButton, AccordionModule, Dialog, TabsModule, 
    Slider, ProgressBar, TreeModule, Rating, ToggleSwitch, DatePicker, Badge, Avatar,
    Tag, Card, Tooltip, Carousel, PickList, OrderList, TreeTableModule, Panel, Fieldset, Splitter, Knob,
    ComplexFeatureComponent, LibraryTestComponent
  ],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App {
  constructor() {
    // @ts-ignore
    console.log('NG_GO_VERSION from define:', process.env.NG_GO_VERSION);
  }

  protected readonly title = signal('new-demo-app');
  activeView: DemoView = 'knob' as any; // default to knob to see it immediately!
  knobValue = 60;
  customerName = 'Go compiler';
  loadDeferred = false;;
  
  checked: boolean = false;
  selectedCity: any | undefined;
  cities = [
      { name: 'New York', code: 'NY' },
      { name: 'Rome', code: 'RM' },
      { name: 'London', code: 'LDN' },
      { name: 'Istanbul', code: 'IST' },
      { name: 'Paris', code: 'PRS' }
  ];
  ingredient: string = '';
  toggleValue: boolean = false;
  
  // Dialog state
  displayDialog: boolean = false;

  // Slider & Progress
  sliderValue: number = 50;
  
  // Extra components
  ratingValue: number = 3;
  switchValue: boolean = false;
  dateValue: Date | null = null;
  
  // Tree state
  treeData: TreeNode[] = [
    {
      key: '0',
      label: 'Documents',
      data: 'Documents Folder',
      icon: 'pi pi-fw pi-inbox',
      children: [
        {
          key: '0-0',
          label: 'Work',
          data: 'Work Folder',
          icon: 'pi pi-fw pi-cog',
          children: [
            { key: '0-0-0', label: 'Expenses.doc', icon: 'pi pi-fw pi-file', data: 'Expenses Document' },
            { key: '0-0-1', label: 'Resume.doc', icon: 'pi pi-fw pi-file', data: 'Resume Document' }
          ]
        },
        {
          key: '0-1',
          label: 'Home',
          data: 'Home Folder',
          icon: 'pi pi-fw pi-home',
          children: [
            { key: '0-1-0', label: 'Invoices.txt', icon: 'pi pi-fw pi-file', data: 'Invoices for month' }
          ]
        }
      ]
    },
    {
      key: '1',
      label: 'Events',
      data: 'Events Folder',
      icon: 'pi pi-fw pi-calendar',
      children: [
        { key: '1-0', label: 'Meeting', icon: 'pi pi-fw pi-calendar-plus', data: 'Meeting' },
        { key: '1-1', label: 'Product Launch', icon: 'pi pi-fw pi-calendar-plus', data: 'Product Launch' },
        { key: '1-2', label: 'Report Review', icon: 'pi pi-fw pi-calendar-plus', data: 'Report Review' }
      ]
    }
  ];

  products: Product[] = [
    { code: 'P-100', name: 'Keyboard', category: 'Accessories', quantity: 24 },
    { code: 'P-101', name: 'Monitor', category: 'Display', quantity: 12 },
    { code: 'P-102', name: 'Notebook', category: 'Office', quantity: 48 },
    { code: 'P-103', name: 'Mouse', category: 'Accessories', quantity: 15 },
    { code: 'P-104', name: 'Headset', category: 'Accessories', quantity: 8 },
  ];
  
  sourceCities: any[] = [
      { name: 'San Francisco', code: 'SF' },
      { name: 'London', code: 'LDN' },
      { name: 'Paris', code: 'PRS' },
      { name: 'Istanbul', code: 'IST' },
      { name: 'Berlin', code: 'BRL' }
  ];
  targetCities: any[] = [];
  
  treeTableData: TreeNode[] = [
      {
          data: { name: 'Applications', size: '200mb', type: 'Folder' },
          children: [
              { data: { name: 'Angular', size: '25mb', type: 'Folder' } },
              { data: { name: 'editor.app', size: '25mb', type: 'Application' } }
          ]
      }
  ];

  items: MenuItem[] = [
      { label: 'Button', icon: 'pi pi-fw pi-box', command: () => this.activeView = 'button' },
      { label: 'Input', icon: 'pi pi-fw pi-pencil', command: () => this.activeView = 'input' },
      { label: 'Form Elements', icon: 'pi pi-fw pi-check-square', command: () => this.activeView = 'form' },
      { label: 'Table', icon: 'pi pi-fw pi-table', command: () => this.activeView = 'table' },
      { label: 'Accordion', icon: 'pi pi-fw pi-list', command: () => this.activeView = 'accordion' },
      { label: 'Dialog', icon: 'pi pi-fw pi-window-maximize', command: () => this.activeView = 'dialog' },
      { label: 'Tabs', icon: 'pi pi-fw pi-clone', command: () => this.activeView = 'tabs' },
      { label: 'Misc', icon: 'pi pi-fw pi-sliders-h', command: () => this.activeView = 'misc' },
      { label: 'Tree', icon: 'pi pi-fw pi-sitemap', command: () => this.activeView = 'tree' },
      { label: 'Extra', icon: 'pi pi-fw pi-plus', command: () => this.activeView = 'extra' },
      { label: 'Advanced', icon: 'pi pi-fw pi-star', command: () => this.activeView = 'advanced' },
      { label: 'Defer', icon: 'pi pi-fw pi-clock', command: () => this.activeView = 'defer' },
      { label: 'Angular Syntax', icon: 'pi pi-fw pi-code', command: () => this.activeView = 'syntax' },
      { label: 'Complex Features', icon: 'pi pi-fw pi-cog', command: () => this.activeView = 'complex' },
      { label: 'External Libraries', icon: 'pi pi-fw pi-box', command: () => this.activeView = 'library' }
  ];

  // Syntax showcase properties
  itemsList = ['Apple', 'Banana', 'Cherry'];
  emptyList: string[] = [];
  userStatus = 'active';
  currentTime = new Date();
  jsonData = { key: 'value', numbers: [1, 2, 3] };
  counter = signal(0);

  increment() {
    this.counter.update(c => c + 1);
  }
}
// test invalidate
// test phase 8
// test phase 8 final
// trigger HMR 185

// trigger hmr

// trigger hmr 2

// trigger hmr 3

// trigger hmr 4

// trigger hmr 5
