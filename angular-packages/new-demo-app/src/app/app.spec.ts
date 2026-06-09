import { renderComponent, GoTestBed } from '@angular-go/build/testbed';
import { provideRouter } from '@angular/router';
import { App } from './app';

describe('App', () => {
  it('should create the app', async () => {
    const fixture = await renderComponent(App, {
      providers: [provideRouter([])]
    });
    expect(fixture.component).toBeTruthy();
  });

  it('should render title', async () => {
    const fixture = await renderComponent(App, {
      providers: [provideRouter([])]
    });
    await fixture.whenStable();
    const compiled = fixture.nativeElement;
    expect(compiled.querySelector('h1')?.textContent).toContain('HMR WORKS BABY AMAZING 24');
  });

  describe('GoTestBed Compatibility', () => {
    beforeEach(async () => {
      GoTestBed.resetTestingModule();
      await GoTestBed.configureTestingModule({
        imports: [App],
        providers: [provideRouter([])]
      }).compileComponents();
    });

    it('should create the component using GoTestBed', () => {
      const fixture = GoTestBed.createComponent(App);
      fixture.detectChanges();
      expect(fixture.componentInstance).toBeTruthy();
      expect(fixture.nativeElement.querySelector('h1')?.textContent).toContain('HMR WORKS BABY AMAZING 24');
    });
  });
});
