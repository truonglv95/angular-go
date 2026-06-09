import { Component, Directive, Input } from '@angular/core';

@Directive({
    selector: '[tooltip]',
    standalone: true
})
export class TooltipDirective {
    @Input() tooltip: string = '';
    
    constructor() {
        console.log('TooltipDirective instantiated');
    }
}

@Directive({
    selector: '[color]',
    standalone: true
})
export class ColorDirective {
    @Input() color: string = 'red';
    
    constructor() {
        console.log('ColorDirective instantiated');
    }
}

@Component({
    selector: 'host-directive-demo',
    standalone: true,
    template: `
        <div class="demo-box">
            Testing HostDirectives:
            <p>Tooltip will be passed to TooltipDirective</p>
            <p>Color will be passed to ColorDirective</p>
        </div>
    `,
    hostDirectives: [
        {
            directive: TooltipDirective,
            inputs: ['tooltip']
        },
        {
            directive: ColorDirective,
            inputs: ['color']
        }
    ]
})
export class HostDirectiveDemoComponent {
}
