import { Component } from 'preact';
import createFocusTrap from 'focus-trap';
import style from './wait-dialog.css';

export default class WaitDialog extends Component {
    render({ active }, {}) {
        const isActive = active ? 'active' : '';
        return (
            <div class={['overlay', isActive].join(' ')}>
                <div class={['modal', isActive].join(' ')}>
                    <h3 class="modalHeader">{this.props.title}</h3>
                    {
                        this.props.children.length > 0 ? (
                            <div class="flex flex-column flex-start">
                                {this.props.children}
                            </div>
                        ) : (
                            <div class="flex flex-column flex-start">
                                <p class={['label', style.confirmationLabel].join(' ')}>On your device</p>
                                <div class={['flex', 'flex-row', 'flex-around', 'flex-items-end', style.confirmationInstructions].join(' ')}>
                                    <div class="flex flex-column flex-center flex-items-center">
                                        <div class={style.shortTouch}></div>
                                        <p class="text-bold">Tap to <span class="text-red">abort</span></p>
                                    </div>
                                    <div class="flex flex-column flex-center flex-items-center">
                                        <div class={style.longTouch}></div>
                                        <p class="text-bold">Hold 3+ secs to <span class="text-green">confirm</span></p>
                                    </div>
                                </div>
                            </div>
                        )
                    }
                </div>
            </div>
        );
    }
}
